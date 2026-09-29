-- Offline harness: loads Parley.lua with a stubbed WoW API, encodes payloads
-- and writes them as PPM images the Go decoder test reads back.
-- usage: lua5.1 gen.lua <outdir>
local out = arg[1] or "."

local function stubFrame()
  local f = {}
  setmetatable(f, { __index = function() return function() return f end end })
  f.CreateTexture = function() return stubFrame() end
  return f
end
CreateFrame = function() return stubFrame() end
UIParent = stubFrame()
DEFAULT_CHAT_FRAME = { AddMessage = function() end }
SlashCmdList = {}

local chunk = assert(loadfile("../addon/Parley/Parley.lua"))
chunk("Parley", {})
local T = Parley_Test

local function writePPM(path, levels, cell, gamma, ox, oy, W, H)
  local f = assert(io.open(path, "wb"))
  f:write(("P6\n%d %d\n255\n"):format(W, H))
  local buf = {}
  for y = 0, H - 1 do
    for x = 0, W - 1 do
      local cx, cy = x - ox, y - oy
      local r, g, b = 40, 60, 80 -- "game" background
      if cx >= 0 and cy >= 0 and cx < T.COLS * cell and cy < T.ROWS * cell then
        local i = math.floor(cy / cell) * T.COLS + math.floor(cx / cell)
        r, g, b = levels[i * 3] * 17, levels[i * 3 + 1] * 17, levels[i * 3 + 2] * 17
        local function gm(v) return math.floor(255 * (v / 255) ^ gamma + 0.5) end
        r, g, b = gm(r), gm(g), gm(b)
      end
      buf[#buf + 1] = string.char(r, g, b)
    end
  end
  f:write(table.concat(buf))
  f:close()
end

local cases = {
  { "W\031\031\031Carlos-Whitemane\031¡Hola! ¿Quieres hacer Deadmines conmigo?", 2, 1.0 },
  { "C\0314\031LookingForGroup\031Иван\031Ищу группу в Мертвые копи, нужен танк", 1, 1.0 },
  { "G\031\031\031小明\031有人去死亡矿井吗？需要坦克和治疗", 3, 1.35 },
  { "S\031\031\031Zed\031" .. T.clean("buy |cff1eff00|Hitem:2140::::::::|h[Carving Knife]|h|r pls || thx"), 2, 0.8 },
  { "X\031\031\031\031", 2, 1.0 }, -- Retail: chat hidden by Blizzard
  { T.cutUtf8("P\031\031\031Max\031" .. string.rep("é", 400), T.MAXBYTES), 4, 1.0 },
}

local manifest = assert(io.open(out .. "/cases.txt", "wb"))
for n, c in ipairs(cases) do
  local levels = T.encode(c[1], n * 7)
  writePPM(("%s/case%d.ppm"):format(out, n), levels, c[2], c[3], 0, 0, 400, 60)
  -- expected payload hex, cell size, sequence
  manifest:write(("%d %d %d %s\n"):format(n, c[2], n * 7, (c[1]:gsub(".", function(ch)
    return ("%02x"):format(ch:byte()) end))))
end
manifest:close()
print("max bytes", T.MAXBYTES, "clean test:", T.clean("buy |cff1eff00|Hitem:2140::::::::|h[Carving Knife]|h|r pls || thx"))
