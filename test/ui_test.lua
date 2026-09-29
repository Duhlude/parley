-- Smoke test for the addon's settings UI and presets with a stubbed WoW API.
-- usage (from the test folder): lua5.1 ui_test.lua
local frames, created = {}, 0
local function newObj(kind)
  local o = { _scripts = {}, _shown = kind ~= "Frame-hidden", _checked = false, _text = "", _kind = kind }
  created = created + 1
  return setmetatable(o, { __index = function(t, k)
    return function(self, ...) return nil end
  end })
end
local function stub(kind, name)
  local o = newObj(kind)
  o.SetScript = function(self, ev, fn) self._scripts[ev] = fn end
  o.GetScript = function(self, ev) return self._scripts[ev] end
  o.Show = function(self) self._shown = true; if self._scripts.OnShow then self._scripts.OnShow(self) end end
  o.Hide = function(self) self._shown = false end
  o.IsShown = function(self) return self._shown end
  o.SetShown = function(self, v) self._shown = v end
  o.SetChecked = function(self, v) self._checked = v and true or false end
  o.GetChecked = function(self) return self._checked end
  o.SetText = function(self, t) self._text = t end
  o.GetText = function(self) return self._text end
  o.GetWidth = function() return 140 end
  o.GetCenter = function() return 100, 100 end
  o.GetEffectiveScale = function() return 1 end
  o.CreateTexture = function() return stub("Texture") end
  o.CreateFontString = function() return stub("FontString") end
  o.StartMoving = function() end
  o.StopMovingOrSizing = function() end
  if kind == "Frame" then o.TitleText = newObj("FontString"); o.TitleText.SetText = function() end end
  frames[#frames + 1] = o
  if name then _G[name] = o end
  return o
end
CreateFrame = function(kind, name, parent, template) return stub(kind, name) end
UIParent, Minimap, GameTooltip = stub("Frame"), stub("Frame"), stub("Frame")
DEFAULT_CHAT_FRAME = { AddMessage = function(_, m) print("chat:", (m:gsub("|c%x%x%x%x%x%x%x%x", ""):gsub("|r", ""))) end }
SlashCmdList, UISpecialFrames = {}, {}
tinsert = table.insert
GetCursorPosition = function() return 150, 100 end
GetPhysicalScreenSize = function() return 1920, 1080 end
GetScreenHeight = function() return 768 end
UnitName = function() return "Max" end
local now = 1000
GetTime = function() return now end
-- Retail secret values: a table stands in for a secret string
issecretvalue = function(v) return type(v) == "table" and v.secret == true end
Settings = { RegisterCanvasLayoutCategory = function() return {} end, RegisterAddOnCategory = function() end }

local ns = {}
assert(loadfile("../addon/Parley/Locale.lua"))("Parley", ns)
assert(ns.T("Close") == "Close", "enUS keeps English")
assert(loadfile("../addon/Parley/Parley.lua"))("Parley", ns)
assert(loadfile("../addon/Parley/ParleyUI.lua"))("Parley", ns)

-- fire ADDON_LOADED + PLAYER_LOGIN on the event frame
for _, f in ipairs(frames) do
  if f._scripts.OnEvent then
    f._scripts.OnEvent(f, "ADDON_LOADED", "Parley")
    f._scripts.OnEvent(f, "PLAYER_LOGIN")
  end
end
local db = ns.getDB()
assert(db and db.minimap and db.customPresets, "defaults")

-- Retail: chat Blizzard hides from addons -> one "X" note, no error
local evFrame
for _, f in ipairs(frames) do if f._scripts.OnEvent then evFrame = f end end
local function pendingCount(prefix)
  local n = 0
  for _, p in ipairs(ns.pending()) do if p:sub(1, #prefix) == prefix then n = n + 1 end end
  return n
end
local secret = { secret = true }
evFrame._scripts.OnEvent(evFrame, "CHAT_MSG_PARTY", secret, secret)
evFrame._scripts.OnEvent(evFrame, "CHAT_MSG_PARTY", secret, secret)
assert(pendingCount("X\031") == 1, "one hidden-chat note per 30 s")
now = now + 31
evFrame._scripts.OnEvent(evFrame, "CHAT_MSG_SAY", "Hola amigos", secret)
assert(pendingCount("X\031") == 2, "note again after 30 s")
evFrame._scripts.OnEvent(evFrame, "CHAT_MSG_SAY", "Hola amigos", "Carlos-Stormrage")
assert(pendingCount("S\031") == 1, "readable chat still goes through")
print("secret chat ok")

-- /parley opens the window
SlashCmdList.PARLEY("")
assert(ParleyConfigFrame and ParleyConfigFrame:IsShown(), "config window shown")

-- click every built-in preset button and check the result
for _, f in ipairs(frames) do
  if rawget(f, "entry") then
    f._scripts.OnClick(f)
    local name = ns.activePresetName()
    assert(name == f.entry.name, "preset " .. f.entry.name .. " -> " .. tostring(name))
  end
end
print("built-in presets ok")

-- untick "Guild" through its checkbox -> settings become custom
ns.applyPreset(ns.PRESETS[1].p)
local guildBox
for _, f in ipairs(frames) do
  if f._kind == "CheckButton" and rawget(f, "label") and f.label._text == "Guild & Officer" then guildBox = f end
end
assert(guildBox, "guild checkbox")
guildBox:SetChecked(false); guildBox._scripts.OnClick(guildBox)
assert(db.groups.guild == false and ns.activePresetName() == nil, "custom after manual change")

-- save it as "Raid night" through the name box + button
local box, saveBtn
for _, f in ipairs(frames) do
  if f._kind == "EditBox" then box = f end
  if f._kind == "Button" and f._text == "Save current as preset" then saveBtn = f end
end
box:SetText("Raid night"); saveBtn._scripts.OnClick(saveBtn)
assert(db.customPresets["Raid night"], "saved")
assert(ns.activePresetName() == "Raid night", "active custom")

-- switch away and back with /parley preset
SlashCmdList.PARLEY("preset group")
assert(ns.activePresetName() == "Group only")
SlashCmdList.PARLEY("preset raid night")
assert(ns.activePresetName() == "Raid night", "custom via slash")
SlashCmdList.PARLEY("preset")
assert(ns.activePresetName() == "Friends & group", "default preset")
SlashCmdList.PARLEY("all")
assert(ns.activePresetName() == "Everything")

-- delete the custom preset with its X button
for _, f in ipairs(frames) do
  if f._kind == "Button" and f._text == "X" and f._scripts.OnClick then f._scripts.OnClick(f); break end
end
assert(next(db.customPresets) == nil, "deleted")

-- minimap button: drag and right-click
assert(ParleyMinimapButton, "minimap button")
ParleyMinimapButton._scripts.OnDragStart(ParleyMinimapButton)
ParleyMinimapButton._scripts.OnUpdate()
ParleyMinimapButton._scripts.OnClick(ParleyMinimapButton, "RightButton")
assert(db.enabled == false, "right-click pauses")
ParleyMinimapButton._scripts.OnEnter(ParleyMinimapButton)
SlashCmdList.PARLEY("status")
SlashCmdList.PARLEY("test")
assert(ns.sendTest("all") == 8, "all samples")
assert(ns.sendTest("asia") == 3, "asia samples")
assert(ns.sendTest("JA") == 1, "one language")
assert(ns.sendTest("xx") == 0, "unknown language")
SlashCmdList.PARLEY("test ja")
SlashCmdList.PARLEY("test nope")
-- other client languages
for loc, want in pairs({ deDE = "Schließen", ptBR = "Fechar", esMX = "Cerrar", ruRU = "Закрыть", zhTW = "關閉" }) do
  GetLocale = function() return loc end
  local ns2 = {}
  assert(loadfile("../addon/Parley/Locale.lua"))("Parley", ns2)
  assert(ns2.T("Close") == want, loc .. " Close = " .. tostring(ns2.T("Close")))
  assert(ns2.T("sent %d test messages."):format(8):find("8"), loc .. " format")
end
print("ALL UI TESTS PASSED")
