// parley-mt: a small persistent translation process for Parley.
//
// Wraps bergamot-translator (Firefox Translations' engine) behind a simple
// length-prefixed protocol on stdin/stdout, so the Parley app can load a model
// once and translate chat lines quickly.
//
// Request  (one header line, fields separated by TAB, then <n> payload bytes):
//   T <id> <html:0|1> <modelDir> <pivotDir or -> <n>\n<utf-8 text>
//   D <id> <n>\n<utf-8 text>                    detect language (CLD2)
//   Q                                          quit
// Response:
//   OK <id> <n>\n<utf-8 text>   (for D: "<code> <percent> <code> <percent> <code> <percent> <reliable 0|1>")
//   ERR <id> <n>\n<message>
//
// A model directory holds the files of one language pair:
//   model.*.bin, vocab.*.spm (or srcvocab.*.spm + trgvocab.*.spm), lex.*.bin
//
// Released under the MIT License as part of Parley.

#include <algorithm>
#include <cstdlib>
#include <cstdio>
#include <filesystem>
#include <iostream>
#include <list>
#include <map>
#include <memory>
#include <sstream>
#include <string>
#include <vector>

#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#include <windows.h>
#endif

#include "public/compact_lang_det.h"
#include "translator/parser.h"
#include "translator/response.h"
#include "translator/response_options.h"
#include "translator/service.h"
#include "translator/translation_model.h"

using namespace marian::bergamot;
namespace fs = std::filesystem;

static std::string findFile(const fs::path &dir, const std::string &prefix, const std::string &ext) {
  std::vector<std::string> hits;
  for (auto &e : fs::directory_iterator(dir)) {
    if (!e.is_regular_file()) continue;
    std::string name = e.path().filename().string();
    if (name.rfind(prefix, 0) == 0 && name.size() >= ext.size() &&
        name.compare(name.size() - ext.size(), ext.size(), ext) == 0)
      hits.push_back(e.path().string());
  }
  std::sort(hits.begin(), hits.end());
  return hits.empty() ? std::string() : hits.front();
}

static std::string yamlQuote(const std::string &s) {
  std::string out = "\"";
  for (char c : s) {
    if (c == '\\' || c == '"') out += '\\';
    out += c;
  }
  return out + "\"";
}

static std::string configFor(const std::string &dirStr) {
  fs::path dir = fs::u8path(dirStr);
  std::string model = findFile(dir, "model.", ".bin");
  std::string src = findFile(dir, "srcvocab.", ".spm");
  std::string trg = findFile(dir, "trgvocab.", ".spm");
  if (src.empty() || trg.empty()) src = trg = findFile(dir, "vocab.", ".spm");
  std::string lex = findFile(dir, "lex.", ".bin");
  if (model.empty() || src.empty()) throw std::runtime_error("model files missing in " + dirStr);

  std::ostringstream c;
  c << "models:\n  - " << yamlQuote(model) << "\n"
    << "vocabs:\n  - " << yamlQuote(src) << "\n  - " << yamlQuote(trg) << "\n";
  if (!lex.empty()) c << "shortlist:\n  - " << yamlQuote(lex) << "\n  - false\n";
  c << "beam-size: 1\n"
       "normalize: 1.0\n"
       "word-penalty: 0\n"
       "max-length-break: 128\n"
       "mini-batch-words: 512\n"
       "max-length-factor: 2.0\n"
       "skip-cost: true\n"
       "cpu-threads: 0\n"
       "quiet: true\n"
       "quiet-translation: true\n"
       "gemm-precision: int8shiftAlphaAll\n"
       "alignment: soft\n";
  // Scratch memory per model. Chat lines are short, so far less than the
  // 128 MB Firefox reserves for whole web pages is enough.
  const char *ws = std::getenv("PARLEY_MT_WORKSPACE");
  c << "workspace: " << (ws ? ws : "40") << "\n";
  return c.str();
}

// Keeps the most recently used models loaded (each takes ~200 MB of RAM).
class ModelCache {
 public:
  explicit ModelCache(size_t max) : max_(max) {}
  std::shared_ptr<TranslationModel> get(const std::string &dir) {
    auto it = map_.find(dir);
    if (it != map_.end()) {
      order_.remove(dir);
      order_.push_front(dir);
      return it->second;
    }
    auto options = parseOptionsFromString(configFor(dir), /*validate=*/false);
    auto model = std::make_shared<TranslationModel>(options, /*replicas=*/1);
    map_[dir] = model;
    order_.push_front(dir);
    while (order_.size() > max_) {
      map_.erase(order_.back());
      order_.pop_back();
    }
    return model;
  }

 private:
  size_t max_;
  std::list<std::string> order_;
  std::map<std::string, std::shared_ptr<TranslationModel>> map_;
};

static void reply(const char *status, const std::string &id, const std::string &body) {
  std::cout << status << '\t' << id << '\t' << body.size() << '\n' << body;
  std::cout.flush();
}

static std::vector<std::string> split(const std::string &s, char sep) {
  std::vector<std::string> out;
  std::string cur;
  for (char c : s) {
    if (c == sep) {
      out.push_back(cur);
      cur.clear();
    } else if (c != '\r') {
      cur += c;
    }
  }
  out.push_back(cur);
  return out;
}

int main(int argc, char *argv[]) {
#ifdef _WIN32
  _setmode(_fileno(stdin), _O_BINARY);
  _setmode(_fileno(stdout), _O_BINARY);
  // Stay out of the game's way.
  SetPriorityClass(GetCurrentProcess(), BELOW_NORMAL_PRIORITY_CLASS);
#endif
  std::ios::sync_with_stdio(false);

  BlockingService::Config serviceConfig;
  serviceConfig.cacheSize = 256;
  serviceConfig.logger.level = "off";
  BlockingService service(serviceConfig);
  ModelCache cache(3);  // enough for a pivot (xx→en→yy) plus one more

  std::cout << "READY\tparley-mt\t1\n";
  std::cout.flush();

  std::string header;
  while (std::getline(std::cin, header)) {
    if (header.empty()) continue;
    auto f = split(header, '\t');
    if (f[0] == "Q") break;
    if (f[0] == "D" && f.size() >= 3) {
      size_t n = std::stoul(f[2]);
      std::string text(n, '\0');
      if (n > 0 && !std::cin.read(&text[0], n)) break;
      CLD2::Language langs[3];
      int percents[3];
      int textBytes = 0;
      bool reliable = false;
      CLD2::ExtDetectLanguageSummary(text.data(), (int)text.size(), true, langs, percents, &textBytes, &reliable);
      std::ostringstream o;
      for (int i = 0; i < 3; i++) o << CLD2::LanguageCode(langs[i]) << ' ' << percents[i] << ' ';
      o << (reliable ? 1 : 0);
      reply("OK", f[1], o.str());
      continue;
    }
    if (f[0] != "T" || f.size() < 6) {
      reply("ERR", f.size() > 1 ? f[1] : "?", "bad request");
      continue;
    }
    const std::string &id = f[1];
    bool html = f[2] == "1";
    size_t n = std::stoul(f[5]);
    std::string text(n, '\0');
    if (n > 0 && !std::cin.read(&text[0], n)) break;

    try {
      auto first = cache.get(f[3]);
      ResponseOptions opts;
      opts.HTML = html;
      std::vector<std::string> sources{std::move(text)};
      std::vector<ResponseOptions> options{opts};
      std::vector<Response> out;
      if (f[4] != "-" && !f[4].empty()) {
        auto second = cache.get(f[4]);
        out = service.pivotMultiple(first, second, std::move(sources), options);
      } else {
        out = service.translateMultiple(first, std::move(sources), options);
      }
      reply("OK", id, out.empty() ? std::string() : out.front().target.text);
    } catch (const std::exception &e) {
      reply("ERR", id, e.what());
    } catch (...) {
      reply("ERR", id, "unknown error");
    }
  }
  return 0;
}
