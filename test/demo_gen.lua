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
  { "C\0314\031LookingForGroup\031Brunao\031Procurando tank e healer para Deadmines, faltam só dois", 2, 1.0 },
  { "C\0312\031Trade - City\031Gustavo\031Vendendo bolsas de seda, 40 prata cada. Me manda mensagem!", 2, 1.0 },
  { "S\031\031\031Thiago\031Alguém sabe onde fica o mestre de pesca em Ventobravo?", 2, 1.0 },
  { "G\031\031\031Duda\031Boa noite pessoal! Quem vai na raid amanhã?", 2, 1.0 },
  { "P\031\031\031Rafa\031Vou puxar o próximo grupo, esperem a mana do curandeiro", 2, 1.0 },
  { "W\031\031\031Larissa-Whitemane\031Oi! Você ainda vende as bolsas? Quanto custa cada uma?", 2, 1.0 },
}
for n, c in ipairs(cases) do
  local levels = T.encode(c[1], n * 7)
  writePPM(("%s/case%d.ppm"):format(out, n), levels, c[2], c[3], 0, 0, 400, 60)
end
print("ok", #cases)
