-- Parley: hands chat messages to the Parley desktop app for translation.
--
-- WoW addons cannot reach the internet, so this addon paints each chat
-- message as a tiny strip of coloured squares in the top-left corner of the
-- game window. The Parley app reads those pixels back into exact text,
-- translates it and shows it in its overlay. Nothing is sent to or read from
-- the game client other than through Blizzard's normal addon API.
--
-- Strip protocol (must match decode.go in the app):
--   Grid of COLS x ROWS cells, each cell CELL x CELL physical pixels.
--   Every cell carries three 4-bit "nibbles", one per colour channel, drawn
--   as level * 17 (0, 17, 34 ... 255).
--   cells 0-3    marker   (15,0,15) (0,15,0) (15,15,0) (0,0,15)
--   cells 4-19   calibration, cell 4+i = (i,i,i)
--   cell  20     sequence number (12 bits)
--   cell  21     payload length in bytes (12 bits)
--   cells 22-23  Fletcher checksum over the payload (mod 4095), low then high
--   cells 24+    payload bytes, two nibbles per byte (high nibble first),
--                packed three nibbles per cell (R, G, B)
--   Payload: type \031 channelNumber \031 channelName \031 sender \031 text

local ADDON, ns = ...
local COLS, ROWS = 64, 6
local NCELLS = COLS * ROWS
local HEADER = 24
local MAXBYTES = math.floor((NCELLS - HEADER) * 3 / 2)
local SEP = "\031"

local defaults = {
  enabled = true,
  cell = 2,          -- physical pixels per cell
  hold = 0.1,        -- seconds each message stays on screen
  autohide = true,   -- hide the strip when there is nothing to send
  friendsOnly = false, -- only translate whispers from people on your friends list
  customPresets = {},  -- name -> { groups = {...}, friendsOnly = bool }
  minimap = { hide = false, angle = 215 },
  groups = { whisper = true, say = true, party = true, raid = true,
             guild = true, channel = true, instance = true },
}

-- chat event -> { type code, settings group }
local EVENTS = {
  CHAT_MSG_WHISPER             = { "W", "whisper" },
  CHAT_MSG_BN_WHISPER          = { "B", "whisper" },
  CHAT_MSG_SAY                 = { "S", "say" },
  CHAT_MSG_YELL                = { "Y", "say" },
  CHAT_MSG_EMOTE               = { "E", "say" },
  CHAT_MSG_PARTY               = { "P", "party" },
  CHAT_MSG_PARTY_LEADER        = { "P", "party" },
  CHAT_MSG_RAID                = { "R", "raid" },
  CHAT_MSG_RAID_LEADER         = { "R", "raid" },
  CHAT_MSG_RAID_WARNING        = { "R", "raid" },
  CHAT_MSG_GUILD               = { "G", "guild" },
  CHAT_MSG_OFFICER             = { "O", "guild" },
  CHAT_MSG_CHANNEL             = { "C", "channel" },
  CHAT_MSG_BATTLEGROUND        = { "I", "instance" },
  CHAT_MSG_BATTLEGROUND_LEADER = { "I", "instance" },
  CHAT_MSG_INSTANCE_CHAT       = { "I", "instance" },
  CHAT_MSG_INSTANCE_CHAT_LEADER= { "I", "instance" },
}

local db
local strip, driver
local textures = {}
local levels = {}      -- flat: levels[cell*3 + ch], ch 0..2
local queue, qhead, qtail = {}, 1, 0
local seq = 0
local since = 0
local pinned = false   -- /parley show keeps the strip visible
local me

local T = ns.T or function(s) return s end

local function say(msg)
  DEFAULT_CHAT_FRAME:AddMessage("|cff60cdffParley:|r " .. msg)
end

local function copyDefaults(src, dst)
  for k, v in pairs(src) do
    if type(v) == "table" then
      if type(dst[k]) ~= "table" then dst[k] = {} end
      copyDefaults(v, dst[k])
    elseif dst[k] == nil then
      dst[k] = v
    end
  end
  return dst
end

---------------------------------------------------------------------------
-- Strip layout: size the frame so one of its units is one physical pixel.
---------------------------------------------------------------------------
local function layout()
  if not strip then return end
  local _, physH = GetPhysicalScreenSize()
  if not physH or physH <= 0 then return end
  -- UIParent height in its own units times its effective scale is the
  -- screen height in base units (768); divide by the real pixel height.
  local base = GetScreenHeight() * UIParent:GetEffectiveScale()
  strip:SetScale(base / physH)

  local c = db.cell
  strip:SetSize(COLS * c, ROWS * c)
  strip:ClearAllPoints()
  strip:SetPoint("TOPLEFT", UIParent, "TOPLEFT", 0, 0)
  for i = 0, NCELLS - 1 do
    local t = textures[i]
    t:SetSize(c, c)
    t:ClearAllPoints()
    t:SetPoint("TOPLEFT", strip, "TOPLEFT", (i % COLS) * c, -math.floor(i / COLS) * c)
  end
end

local function buildStrip()
  strip = CreateFrame("Frame", "ParleyStrip", nil)
  strip:SetFrameStrata("TOOLTIP")
  strip:SetFrameLevel(9000)
  strip:SetClampedToScreen(false)
  strip:EnableMouse(false)
  for i = 0, NCELLS - 1 do
    local t = strip:CreateTexture(nil, "OVERLAY")
    if t.SetSnapToPixelGrid then t:SetSnapToPixelGrid(false) end
    if t.SetTexelSnappingBias then t:SetTexelSnappingBias(0) end
    t:SetColorTexture(0, 0, 0, 1)
    textures[i] = t
  end
  strip:Hide()
  layout()
end

---------------------------------------------------------------------------
-- Encoding
---------------------------------------------------------------------------
local function setCell(i, r, g, b)
  levels[i * 3], levels[i * 3 + 1], levels[i * 3 + 2] = r, g, b
end

local function set12(i, v)
  setCell(i, math.floor(v / 256) % 16, math.floor(v / 16) % 16, v % 16)
end

local function fletcher(s)
  local a, b = 0, 0
  for i = 1, #s do
    a = (a + s:byte(i)) % 4095
    b = (b + a) % 4095
  end
  return a, b
end

-- Fill `levels` with a full frame for payload `s`. Exposed for tests.
local function encode(s, sequence)
  for i = 0, NCELLS * 3 - 1 do levels[i] = 0 end
  setCell(0, 15, 0, 15); setCell(1, 0, 15, 0)
  setCell(2, 15, 15, 0); setCell(3, 0, 0, 15)
  for i = 0, 15 do setCell(4 + i, i, i, i) end
  local a, b = fletcher(s)
  set12(20, sequence); set12(21, #s); set12(22, a); set12(23, b)
  local n = HEADER * 3
  for i = 1, #s do
    local byte = s:byte(i)
    levels[n] = math.floor(byte / 16); levels[n + 1] = byte % 16
    n = n + 2
  end
  return levels
end

local function paint()
  for i = 0, NCELLS - 1 do
    textures[i]:SetColorTexture(levels[i * 3] / 15, levels[i * 3 + 1] / 15,
                                levels[i * 3 + 2] / 15, 1)
  end
end

---------------------------------------------------------------------------
-- Text cleanup
---------------------------------------------------------------------------
local function clean(s)
  if type(s) ~= "string" then return "" end
  s = s:gsub("||", "\001")
  s = s:gsub("|c%x%x%x%x%x%x%x%x", ""):gsub("|r", "")
  s = s:gsub("|H.-|h(.-)|h", "%1")
  s = s:gsub("|T.-|t", ""):gsub("|A.-|a", "")
  s = s:gsub("|K.-|k", "?")
  s = s:gsub("|n", " ")
  s = s:gsub("\001", "|")
  s = s:gsub(SEP, " ")
  return s
end

-- Cut a string to at most n bytes without splitting a UTF-8 character.
local function cutUtf8(s, n)
  if #s <= n then return s end
  s = s:sub(1, n)
  while #s > 0 do
    local last = s:byte(#s)
    if last < 128 then break end
    -- walk back to the lead byte of the final character
    local i = #s
    while i > 1 and s:byte(i) >= 128 and s:byte(i) < 192 do i = i - 1 end
    local lead = s:byte(i)
    local need = (lead >= 240 and 4) or (lead >= 224 and 3) or (lead >= 192 and 2) or 1
    if #s - i + 1 >= need then break end
    s = s:sub(1, i - 1)
  end
  return s
end

---------------------------------------------------------------------------
-- Queue + driver (OnUpdate only runs while there is work)
---------------------------------------------------------------------------
local function enqueue(payload)
  qtail = qtail + 1
  queue[qtail] = payload
  if qtail - qhead + 1 > 60 then           -- drop oldest if flooded
    queue[qhead] = nil
    qhead = qhead + 1
  end
  driver:Show()
end

local function onUpdate(_, elapsed)
  since = since + elapsed
  if qhead <= qtail then
    if since >= db.hold then
      local payload = queue[qhead]
      queue[qhead] = nil
      qhead = qhead + 1
      seq = (seq + 1) % 4096
      encode(payload, seq)
      paint()
      strip:Show()
      since = 0
    end
  elseif since >= db.hold + 0.4 then
    if db.autohide and not pinned then strip:Hide() end
    driver:Hide()
    qhead, qtail = 1, 0
  end
end

local function send(code, chanNum, chanName, sender, text)
  local head = table.concat({ code, tostring(chanNum or ""), clean(chanName or ""),
                              clean(sender or ""), "" }, SEP)
  local payload = cutUtf8(head .. clean(text), MAXBYTES)
  enqueue(payload)
end

---------------------------------------------------------------------------
-- Events
---------------------------------------------------------------------------
-- Is the sender on your friends list (character or Battle.net friend)?
local function isFriend(sender, guid)
  if guid and C_FriendList and C_FriendList.IsFriend then
    local ok, r = pcall(C_FriendList.IsFriend, guid)
    if ok and r then return true end
  end
  if type(sender) == "string" and C_FriendList and C_FriendList.GetFriendInfo then
    local ok, info = pcall(C_FriendList.GetFriendInfo, (sender:match("^[^-]+")))
    if ok and info then return true end
  end
  if guid and C_BattleNet and C_BattleNet.GetGameAccountInfoByGUID then
    local ok, info = pcall(C_BattleNet.GetGameAccountInfoByGUID, guid)
    if ok and info then return true end
  end
  return false
end

-- Retail (Midnight and later) hands chat to addons as "secret values" during
-- Mythic+ keys, PvP matches and boss fights: they can be shown but not read.
-- Tell the app once in a while instead of erroring on them.
local issecret = issecretvalue or function() return false end
local lastHiddenNote = -100

local function isHidden(...)
  for i = 1, select("#", ...) do
    if issecret((select(i, ...))) then return true end
  end
  return false
end

local function onChat(event, ...)
  if not db.enabled then return end
  local text, sender, _, _, _, _, _, chanNum, chanBase, _, _, guid = ...
  local info = EVENTS[event]
  if not info or not db.groups[info[2]] then return end
  if isHidden(text, sender, chanNum, chanBase, guid) then
    local now = GetTime()
    if now - lastHiddenNote > 30 then
      lastHiddenNote = now
      enqueue(table.concat({ "X", "", "", "", "" }, SEP))
    end
    return
  end
  if type(sender) == "string" and me and sender:match("^[^-]+") == me then return end
  if type(text) ~= "string" or text == "" then return end
  if info[1] == "W" and db.friendsOnly and not isFriend(sender, guid) then return end
  if info[1] == "C" then
    send("C", chanNum, chanBase, sender, text)
  else
    send(info[1], "", "", sender, text)
  end
end

local events = CreateFrame("Frame")
events:RegisterEvent("ADDON_LOADED")
events:RegisterEvent("PLAYER_LOGIN")
events:RegisterEvent("DISPLAY_SIZE_CHANGED")
events:RegisterEvent("UI_SCALE_CHANGED")
for ev in pairs(EVENTS) do
  pcall(events.RegisterEvent, events, ev)  -- some events don't exist on every client
end

events:SetScript("OnEvent", function(_, event, ...)
  if event == "ADDON_LOADED" then
    if ... ~= ADDON then return end
    ParleyDB = copyDefaults(defaults, ParleyDB or {})
    db = ParleyDB
    buildStrip()
    driver = CreateFrame("Frame")
    driver:Hide()
    driver:SetScript("OnUpdate", onUpdate)
    if ns.OnLoaded then ns.OnLoaded() end
  elseif event == "PLAYER_LOGIN" then
    me = UnitName("player")
    layout()
    say(T("loaded. Type /parley for settings."))
  elseif event == "DISPLAY_SIZE_CHANGED" or event == "UI_SCALE_CHANGED" then
    layout()
  elseif db then
    onChat(event, ...)
  end
end)

---------------------------------------------------------------------------
-- Presets
---------------------------------------------------------------------------
local GROUPS = { "whisper", "say", "party", "raid", "guild", "channel", "instance" }

local function preset(friendsOnly, ...)
  local g = {}
  for _, k in ipairs(GROUPS) do g[k] = false end
  for _, k in ipairs({ ... }) do g[k] = true end
  return { groups = g, friendsOnly = friendsOnly }
end

-- Built-in presets, in display order.
local PRESETS = {
  { key = "everything", name = "Everything", desc = "All chat types, including public channels",
    p = preset(false, "whisper", "say", "party", "raid", "guild", "channel", "instance") },
  { key = "friendsgroup", name = "Friends & group", desc = "Whispers from friends, party, raid and battlegrounds",
    p = preset(true, "whisper", "party", "raid", "instance") },
  { key = "group", name = "Group only", desc = "Party, raid and battleground chat",
    p = preset(false, "party", "raid", "instance") },
  { key = "social", name = "Social", desc = "Whispers, say/yell and guild, no public channels",
    p = preset(false, "whisper", "say", "guild") },
  { key = "nearby", name = "Nearby & whispers", desc = "Whispers plus people talking near you",
    p = preset(false, "whisper", "say") },
  { key = "quiet", name = "Friends only", desc = "Only whispers from your friends list",
    p = preset(true, "whisper") },
}

local function applyPreset(p)
  for _, k in ipairs(GROUPS) do db.groups[k] = p.groups[k] and true or false end
  db.friendsOnly = p.friendsOnly and true or false
end

local function matches(p)
  for _, k in ipairs(GROUPS) do
    if (db.groups[k] and true or false) ~= (p.groups[k] and true or false) then return false end
  end
  return (db.friendsOnly and true or false) == (p.friendsOnly and true or false)
end

-- Name of the preset the current settings match, or nil for "custom".
local function activePresetName()
  for _, e in ipairs(PRESETS) do if matches(e.p) then return e.name, false end end
  for name, p in pairs(db.customPresets) do if matches(p) then return name, true end end
  return nil
end

local function saveCustomPreset(name)
  name = (name or ""):gsub("^%s+", ""):gsub("%s+$", "")
  if name == "" then return false, T("type a name first") end
  local count = 0
  for _ in pairs(db.customPresets) do count = count + 1 end
  if not db.customPresets[name] and count >= 8 then return false, T("you can save up to 8 presets") end
  local g = {}
  for _, k in ipairs(GROUPS) do g[k] = db.groups[k] and true or false end
  db.customPresets[name] = { groups = g, friendsOnly = db.friendsOnly and true or false }
  return true
end

---------------------------------------------------------------------------
-- Slash commands
---------------------------------------------------------------------------
local function onOff(v) return v and ("|cff40ff40" .. T("on") .. "|r") or ("|cffff4040" .. T("off") .. "|r") end

local function presetLabel(name, isCustom)
  if not name then return T("custom") end
  return isCustom and name or T(name)
end

local function status()
  say(T("translation feed %s, preset: %s"):format(onOff(db.enabled), presetLabel(activePresetName())))
  local g = {}
  for _, k in ipairs(GROUPS) do g[#g + 1] = k .. " " .. onOff(db.groups[k]) end
  say(table.concat(g, ", ") .. ", " .. T("whispers from friends only %s"):format(onOff(db.friendsOnly)))
  say(T("/parley opens the settings window. Also:") .. " status | on | off | test | show | preset <name> | all")
end

local function findPreset(arg)
  arg = (arg or ""):lower():gsub("%s+", "")
  if arg == "" then return PRESETS[2] end -- plain "/parley preset" = Friends & group
  for _, e in ipairs(PRESETS) do
    if e.key == arg or e.name:lower():gsub("%s+", "") == arg or T(e.name):lower():gsub("%s+", "") == arg then return e end
  end
  for name, p in pairs(db.customPresets) do
    if name:lower():gsub("%s+", "") == arg then return { name = name, p = p, custom = true } end
  end
end

local function togglePin()
  pinned = not pinned
  if pinned then strip:Show() elseif db.autohide then strip:Hide() end
  return pinned
end

local SAMPLES = {
  es = { "W", "", "", "¡Hola! ¿Quieres hacer Deadmines conmigo? Necesitamos un tanque." },
  ru = { "C", "4", "LookingForGroup", "Ищу группу в Мертвые копи, нужен танк и хил" },
  pt = { "G", "", "", "Alguém pode me ajudar com a missão em Westfall?" },
  de = { "P", "", "", "Hat jemand Lust, später Scholomance zu machen? Wir brauchen noch einen Heiler." },
  fr = { "W", "", "", "Salut ! Tu veux venir avec nous au Monastère Écarlate ?" },
  ja = { "W", "", "", "こんにちは！一緒にダンジョンに行きませんか？タンクを探しています。" },
  zh = { "C", "4", "LookingForGroup", "有人去死亡矿井吗？需要一个坦克和一个治疗。" },
  ko = { "P", "", "", "안녕하세요! 같이 던전 가실래요? 힐러가 필요해요." },
}
local SAMPLE_ORDER = { "es", "ru", "pt", "de", "fr", "ja", "zh", "ko" }

-- /parley test [language|all|asia]: sample messages for checking the app.
local function sendTest(which)
  which = (which or ""):lower()
  local list
  if which == "" then list = { "es", "ru", "pt" }
  elseif which == "all" then list = SAMPLE_ORDER
  elseif which == "asia" or which == "cjk" then list = { "ja", "zh", "ko" }
  elseif SAMPLES[which] then list = { which }
  else return 0 end
  for _, code in ipairs(list) do
    local m = SAMPLES[code]
    send(m[1], m[2], m[3], "Parley", m[4])
  end
  return #list
end

SLASH_PARLEY1 = "/parley"
SlashCmdList.PARLEY = function(msg)
  local cmd, arg = (msg or ""):match("^%s*(%S*)%s*(.-)%s*$")
  cmd = cmd:lower()
  if cmd == "" or cmd == "config" or cmd == "options" then
    if ns.ToggleConfig then ns.ToggleConfig() else status() end
  elseif cmd == "status" or cmd == "help" then
    status()
  elseif cmd == "on" or cmd == "off" then
    db.enabled = (cmd == "on"); say(T("translation feed %s"):format(onOff(db.enabled)))
  elseif cmd == "test" then
    local n = sendTest(arg)
    if n == 1 then say(T("sent 1 test message."))
    elseif n > 1 then say(T("sent %d test messages."):format(n))
    else say(T("usage: /parley test [language], where language is es, ru, pt, de, fr, ja, zh, ko, asia or all")) end
  elseif cmd == "show" then
    say(T("strip pinned %s"):format(onOff(togglePin())))
  elseif cmd == "autohide" then
    db.autohide = not db.autohide
    if db.autohide and not pinned and not driver:IsShown() then strip:Hide() else strip:Show() end
    say(T("autohide %s"):format(onOff(db.autohide)))
  elseif cmd == "cell" then
    local n = tonumber(arg)
    if n and n >= 1 and n <= 4 then db.cell = math.floor(n); layout(); say(T("cell size %dpx"):format(db.cell))
    else say(T("cell size must be 1-4")) end
  elseif cmd == "friendsonly" then
    db.friendsOnly = not db.friendsOnly
    say(T("whispers from friends only %s"):format(onOff(db.friendsOnly)))
  elseif cmd == "preset" then
    local e = findPreset(arg)
    if e then applyPreset(e.p); say(T("preset: %s."):format(presetLabel(e.name, e.custom)))
    else say(T("no preset called '%s'."):format(arg)) end
  elseif cmd == "all" then
    applyPreset(PRESETS[1].p); say(T("now translating all chat types."))
  elseif cmd == "hold" then
    local n = tonumber(arg)
    if n and n >= 0.03 and n <= 2 then db.hold = n; say(T("hold %ss"):format(n))
    else say(T("hold must be between 0.03 and 2 seconds")) end
  elseif db.groups[cmd] ~= nil then
    db.groups[cmd] = not db.groups[cmd]; say(cmd .. " " .. onOff(db.groups[cmd]))
  else
    status()
  end
  if ns.RefreshConfig then ns.RefreshConfig() end
end

-- Shared with ParleyUI.lua
ns.GROUPS, ns.PRESETS = GROUPS, PRESETS
ns.applyPreset, ns.activePresetName, ns.saveCustomPreset = applyPreset, activePresetName, saveCustomPreset
ns.togglePin, ns.sendTest, ns.say = togglePin, sendTest, say
ns.isPinned = function() return pinned end
ns.getDB = function() return db end
ns.pending = function()
  local t = {}
  for i = qhead, qtail do t[#t + 1] = queue[i] end
  return t
end

-- Test hook (only used by the offline test harness).
Parley_Test = { encode = encode, clean = clean, cutUtf8 = cutUtf8,
                COLS = COLS, ROWS = ROWS, MAXBYTES = MAXBYTES }
