-- Parley settings window, Interface Options entry and minimap button.
-- Open with /parley, the minimap button, or Esc > Options > AddOns > Parley.

local ADDON, ns = ...

local frame            -- the settings window
local checks = {}      -- chat-type checkboxes by group key
local presetButtons = {}
local customRows = {}
local friendsCheck, enabledCheck, minimapCheck, pinButton, presetLabel, nameBox, saveMsg
local minimapButton

local GROUP_LABELS = {
  { key = "whisper",  label = "Whispers" },
  { key = "say",      label = "Say / Yell / Emote" },
  { key = "party",    label = "Party" },
  { key = "raid",     label = "Raid" },
  { key = "guild",    label = "Guild & Officer" },
  { key = "channel",  label = "Public channels" },
  { key = "instance", label = "Battleground" },
}

local function db() return ns.getDB() end
local T = ns.T or function(s) return s end
local function presetText(name, isCustom)
  if not name then return T("custom") end
  return isCustom and name or T(name)
end

---------------------------------------------------------------------------
-- Small widget helpers
---------------------------------------------------------------------------
local function header(parent, text, x, y)
  local fs = parent:CreateFontString(nil, "ARTWORK", "GameFontNormal")
  fs:SetPoint("TOPLEFT", x, y)
  fs:SetText(text)
  return fs
end

local function note(parent, text, x, y, width)
  local fs = parent:CreateFontString(nil, "ARTWORK", "GameFontHighlightSmall")
  fs:SetPoint("TOPLEFT", x, y)
  if width then fs:SetWidth(width) end
  fs:SetJustifyH("LEFT")
  fs:SetText(text)
  return fs
end

local function checkbox(parent, label, x, y, onClick, tip)
  local cb = CreateFrame("CheckButton", nil, parent, "UICheckButtonTemplate")
  cb:SetSize(24, 24)
  cb:SetPoint("TOPLEFT", x, y)
  local fs = cb:CreateFontString(nil, "ARTWORK", "GameFontHighlight")
  fs:SetPoint("LEFT", cb, "RIGHT", 2, 1)
  fs:SetText(label)
  cb.label = fs
  cb:SetScript("OnClick", function(self)
    onClick(self:GetChecked() and true or false)
    ns.RefreshConfig()
  end)
  if tip then
    cb:SetScript("OnEnter", function(self)
      GameTooltip:SetOwner(self, "ANCHOR_RIGHT")
      GameTooltip:SetText(label, 1, 1, 1)
      GameTooltip:AddLine(tip, nil, nil, nil, true)
      GameTooltip:Show()
    end)
    cb:SetScript("OnLeave", function() GameTooltip:Hide() end)
  end
  return cb
end

-- grow = widen the button to fit longer (translated) text; otherwise the
-- text is shortened with "..." so buttons in a grid never overlap.
local function button(parent, text, w, h, grow)
  local b = CreateFrame("Button", nil, parent, "UIPanelButtonTemplate")
  b:SetSize(w, h or 22)
  b:SetText(text)
  local fs = b.GetFontString and b:GetFontString()
  if fs and fs.GetStringWidth then
    local need = fs:GetStringWidth() + 24
    if need > w then
      if grow then
        b:SetWidth(need)
      elseif fs.SetWordWrap then
        fs:SetWidth(w - 14)
        fs:SetWordWrap(false)
      end
    end
  end
  return b
end

---------------------------------------------------------------------------
-- Settings window
---------------------------------------------------------------------------
local function buildFrame()
  local ok, f = pcall(CreateFrame, "Frame", "ParleyConfigFrame", UIParent, "BasicFrameTemplateWithInset")
  if not ok or not f then
    f = CreateFrame("Frame", "ParleyConfigFrame", UIParent, BackdropTemplateMixin and "BackdropTemplate" or nil)
    if f.SetBackdrop then
      f:SetBackdrop({ bgFile = "Interface\\DialogFrame\\UI-DialogBox-Background",
                      edgeFile = "Interface\\DialogFrame\\UI-DialogBox-Border",
                      tile = true, tileSize = 32, edgeSize = 32,
                      insets = { left = 11, right = 12, top = 12, bottom = 11 } })
    end
    local close = CreateFrame("Button", nil, f, "UIPanelCloseButton")
    close:SetPoint("TOPRIGHT", -4, -4)
  end
  f:SetSize(440, 560)
  f:SetPoint("CENTER")
  f:SetFrameStrata("DIALOG")
  f:SetClampedToScreen(true)
  f:EnableMouse(true)
  f:SetMovable(true)
  f:RegisterForDrag("LeftButton")
  f:SetScript("OnDragStart", f.StartMoving)
  f:SetScript("OnDragStop", f.StopMovingOrSizing)
  f:Hide()
  tinsert(UISpecialFrames, "ParleyConfigFrame") -- Esc closes it

  local title = f.TitleText or f.title
  if not title then
    title = f:CreateFontString(nil, "OVERLAY", "GameFontHighlight")
    title:SetPoint("TOP", 0, -6)
  end
  title:SetText("Parley")

  local L = 16
  -- master switch
  enabledCheck = checkbox(f, T("Send chat to the Parley app"), L, -30, function(v) db().enabled = v end,
    T("Turn off to pause translation without closing the Parley app."))
  note(f, T("The Parley desktop app must be running to translate."), L + 28, -52, 360)

  -- presets
  header(f, T("Presets"), L, -76)
  presetLabel = note(f, "", L + 70, -78, 300)
  for i, e in ipairs(ns.PRESETS) do
    local b = button(f, T(e.name), 186, 24)
    local col, row = (i - 1) % 2, math.floor((i - 1) / 2)
    b:SetPoint("TOPLEFT", L + col * 196, -96 - row * 28)
    b:SetScript("OnClick", function() ns.applyPreset(e.p); ns.RefreshConfig() end)
    b:SetScript("OnEnter", function(self)
      GameTooltip:SetOwner(self, "ANCHOR_RIGHT")
      GameTooltip:SetText(T(e.name), 1, 1, 1)
      GameTooltip:AddLine(T(e.desc), nil, nil, nil, true)
      GameTooltip:Show()
    end)
    b:SetScript("OnLeave", function() GameTooltip:Hide() end)
    b.entry = e
    presetButtons[#presetButtons + 1] = b
  end

  -- my presets
  local myY = -96 - math.ceil(#ns.PRESETS / 2) * 28 - 8
  header(f, T("My presets"), L, myY)
  f.emptyNote = note(f, T("None yet: set the chat types below, name it and click Save."), L, myY - 20, 380)
  for i = 1, 8 do
    local row = CreateFrame("Frame", nil, f)
    row:SetSize(186, 24)
    local col, r = (i - 1) % 2, math.floor((i - 1) / 2)
    row:SetPoint("TOPLEFT", L + col * 196, myY - 20 - r * 26)
    row.apply = button(row, "", 158, 22)
    row.apply:SetPoint("LEFT")
    row.del = button(row, "X", 26, 22)
    row.del:SetPoint("LEFT", row.apply, "RIGHT", 2, 0)
    row:Hide()
    customRows[i] = row
  end
  local saveY = myY - 20 - 4 * 26 - 4
  nameBox = CreateFrame("EditBox", nil, f, "InputBoxTemplate")
  nameBox:SetSize(180, 22)
  nameBox:SetPoint("TOPLEFT", L + 6, saveY)
  nameBox:SetAutoFocus(false)
  nameBox:SetMaxLetters(24)
  local save = button(f, T("Save current as preset"), 180, 22, true)
  save:SetPoint("LEFT", nameBox, "RIGHT", 10, 0)
  local function doSave()
    local ok2, err = ns.saveCustomPreset(nameBox:GetText())
    saveMsg:SetText(ok2 and ("|cff40ff40" .. T("Saved.") .. "|r") or ("|cffff6040" .. err .. "|r"))
    if ok2 then nameBox:SetText(""); nameBox:ClearFocus() end
    ns.RefreshConfig()
  end
  save:SetScript("OnClick", doSave)
  nameBox:SetScript("OnEnterPressed", doSave)
  nameBox:SetScript("OnEscapePressed", function(self) self:ClearFocus() end)
  saveMsg = note(f, "", L + 6, saveY - 26, 380)

  -- chat types
  local ctY = saveY - 48
  header(f, T("Chat types to translate"), L, ctY)
  local tips = {
    channel = T("General, Trade, LocalDefense, LookingForGroup and custom channels."),
    say = T("People talking near you, including custom emotes."),
    instance = T("Battleground chat."),
  }
  local left = { "whisper", "friends", "say", "party" }
  local right = { "raid", "guild", "channel", "instance" }
  local labels = {}
  for _, g in ipairs(GROUP_LABELS) do labels[g.key] = T(g.label) end
  for col, list in ipairs({ left, right }) do
    for row, key in ipairs(list) do
      local x, y = L + (col - 1) * 196, ctY - 20 - (row - 1) * 24
      if key == "friends" then
        friendsCheck = checkbox(f, T("Only from friends"), x + 22, y,
          function(v) db().friendsOnly = v end,
          T("Only translate whispers from your friends list (characters and Battle.net friends)."))
      else
        checks[key] = checkbox(f, labels[key], x, y, function(v) db().groups[key] = v end, tips[key])
      end
    end
  end
  local y = ctY - 20 - 4 * 24

  -- bottom
  minimapCheck = checkbox(f, T("Show minimap button"), L, y - 6, function(v)
    db().minimap.hide = not v
    ns.UpdateMinimap()
  end)
  local test = button(f, T("Send test messages"), 150, 24, true)
  test:SetPoint("BOTTOMLEFT", L, 14)
  test:SetScript("OnClick", function() ns.sendTest(); ns.say(T("sent %d test messages."):format(3)) end)
  pinButton = button(f, T("Show strip"), 110, 24, true)
  pinButton:SetPoint("LEFT", test, "RIGHT", 8, 0)
  pinButton:SetScript("OnClick", function() ns.togglePin(); ns.RefreshConfig() end)
  pinButton:SetScript("OnEnter", function(self)
    GameTooltip:SetOwner(self, "ANCHOR_RIGHT")
    GameTooltip:SetText(T("Show strip"), 1, 1, 1)
    GameTooltip:AddLine(T("Keeps the coloured strip in the top-left corner visible, for troubleshooting."), nil, nil, nil, true)
    GameTooltip:Show()
  end)
  pinButton:SetScript("OnLeave", function() GameTooltip:Hide() end)
  local close = button(f, T("Close"), 90, 24, true)
  close:SetPoint("BOTTOMRIGHT", -16, 14)
  close:SetScript("OnClick", function() f:Hide() end)

  f:SetScript("OnShow", function() ns.RefreshConfig() end)
  frame = f
end

function ns.RefreshConfig()
  if not frame or not frame:IsShown() then return end
  local d = db()
  enabledCheck:SetChecked(d.enabled)
  for key, cb in pairs(checks) do cb:SetChecked(d.groups[key]) end
  friendsCheck:SetChecked(d.friendsOnly)
  if friendsCheck.SetEnabled then friendsCheck:SetEnabled(d.groups.whisper) end
  friendsCheck.label:SetTextColor(d.groups.whisper and 1 or 0.5, d.groups.whisper and 1 or 0.5, d.groups.whisper and 1 or 0.5)
  minimapCheck:SetChecked(not d.minimap.hide)
  pinButton:SetText(ns.isPinned() and T("Hide strip") or T("Show strip"))

  local active, isCustom = ns.activePresetName()
  presetLabel:SetText(active and (T("Active:") .. " |cffffd100" .. presetText(active, isCustom) .. "|r") or (T("Active:") .. " |cffaaaaaa" .. T("custom") .. "|r"))
  for _, b in ipairs(presetButtons) do
    if not isCustom and active == b.entry.name then b:LockHighlight() else b:UnlockHighlight() end
  end

  local names = {}
  for name in pairs(d.customPresets) do names[#names + 1] = name end
  table.sort(names)
  for i, row in ipairs(customRows) do
    local name = names[i]
    if name then
      row.apply:SetText(name)
      row.apply:SetScript("OnClick", function() ns.applyPreset(d.customPresets[name]); ns.RefreshConfig() end)
      row.del:SetScript("OnClick", function() d.customPresets[name] = nil; ns.RefreshConfig() end)
      if isCustom and active == name then row.apply:LockHighlight() else row.apply:UnlockHighlight() end
      row:Show()
    else
      row:Hide()
    end
  end
  frame.emptyNote:SetShown(#names == 0)
end

function ns.ToggleConfig()
  if not frame then buildFrame() end
  if frame:IsShown() then frame:Hide() else frame:Show() end
end

---------------------------------------------------------------------------
-- Interface Options entry (Esc > Options > AddOns > Parley)
---------------------------------------------------------------------------
local function registerOptions()
  local panel = CreateFrame("Frame")
  panel.name = "Parley"
  local t = panel:CreateFontString(nil, "ARTWORK", "GameFontNormalLarge")
  t:SetPoint("TOPLEFT", 16, -16)
  t:SetText("Parley")
  local d = panel:CreateFontString(nil, "ARTWORK", "GameFontHighlight")
  d:SetPoint("TOPLEFT", t, "BOTTOMLEFT", 0, -8)
  d:SetWidth(520)
  d:SetJustifyH("LEFT")
  d:SetText(T("Live chat translation. Presets and chat types are set in the Parley window."))
  local open = CreateFrame("Button", nil, panel, "UIPanelButtonTemplate")
  open:SetSize(200, 26)
  open:SetPoint("TOPLEFT", d, "BOTTOMLEFT", 0, -14)
  open:SetText(T("Open Parley settings"))
  open:SetScript("OnClick", function()
    if SettingsPanel and SettingsPanel:IsShown() then HideUIPanel(SettingsPanel) end
    if InterfaceOptionsFrame and InterfaceOptionsFrame:IsShown() then InterfaceOptionsFrame:Hide() end
    if not frame then buildFrame() end
    frame:Show()
  end)
  if Settings and Settings.RegisterCanvasLayoutCategory and Settings.RegisterAddOnCategory then
    local cat = Settings.RegisterCanvasLayoutCategory(panel, "Parley")
    Settings.RegisterAddOnCategory(cat)
  elseif InterfaceOptions_AddCategory then
    InterfaceOptions_AddCategory(panel)
  end
end

---------------------------------------------------------------------------
-- Minimap button (drag to move around the minimap edge)
---------------------------------------------------------------------------
local function placeMinimapButton()
  local a = math.rad(db().minimap.angle or 215)
  local r = (Minimap:GetWidth() / 2) + 5
  minimapButton:ClearAllPoints()
  minimapButton:SetPoint("CENTER", Minimap, "CENTER", math.cos(a) * r, math.sin(a) * r)
end

local function buildMinimapButton()
  local b = CreateFrame("Button", "ParleyMinimapButton", Minimap)
  b:SetSize(31, 31)
  b:SetFrameStrata("MEDIUM")
  b:SetFrameLevel(8)
  b:RegisterForClicks("LeftButtonUp", "RightButtonUp")
  b:RegisterForDrag("LeftButton")
  b:SetHighlightTexture("Interface\\Minimap\\UI-Minimap-ZoomButton-Highlight")
  local bg = b:CreateTexture(nil, "BACKGROUND")
  bg:SetTexture("Interface\\Minimap\\UI-Minimap-Background")
  bg:SetSize(20, 20)
  bg:SetPoint("TOPLEFT", 7, -5)
  local icon = b:CreateTexture(nil, "ARTWORK")
  icon:SetTexture("Interface\\AddOns\\Parley\\Media\\minimap")
  icon:SetSize(20, 20)
  icon:SetPoint("TOPLEFT", 6, -5)
  local border = b:CreateTexture(nil, "OVERLAY")
  border:SetTexture("Interface\\Minimap\\MiniMap-TrackingBorder")
  border:SetSize(53, 53)
  border:SetPoint("TOPLEFT")
  b:SetScript("OnClick", function(_, btn)
    if btn == "RightButton" then
      local d = db()
      d.enabled = not d.enabled
      ns.say(T("translation feed %s"):format(d.enabled and ("|cff40ff40" .. T("on") .. "|r") or ("|cffff4040" .. T("off") .. "|r")))
      ns.RefreshConfig()
    else
      ns.ToggleConfig()
    end
  end)
  b:SetScript("OnDragStart", function(self)
    self:SetScript("OnUpdate", function()
      local mx, my = Minimap:GetCenter()
      local cx, cy = GetCursorPosition()
      local s = Minimap:GetEffectiveScale()
      db().minimap.angle = math.deg(math.atan2(cy / s - my, cx / s - mx))
      placeMinimapButton()
    end)
  end)
  b:SetScript("OnDragStop", function(self) self:SetScript("OnUpdate", nil) end)
  b:SetScript("OnEnter", function(self)
    GameTooltip:SetOwner(self, "ANCHOR_LEFT")
    GameTooltip:SetText("Parley")
    local d = db()
    GameTooltip:AddLine(T("Preset:") .. " " .. presetText(ns.activePresetName()), 1, 1, 1)
    GameTooltip:AddLine(T("Translation feed:") .. " " .. (d.enabled and ("|cff40ff40" .. T("on") .. "|r") or ("|cffff4040" .. T("off") .. "|r")), 1, 1, 1)
    GameTooltip:AddLine(T("Left-click: settings   Right-click: pause/resume"), 0.7, 0.7, 0.7)
    GameTooltip:Show()
  end)
  b:SetScript("OnLeave", function() GameTooltip:Hide() end)
  minimapButton = b
end

function ns.UpdateMinimap()
  if not minimapButton then buildMinimapButton() end
  if db().minimap.hide then minimapButton:Hide() else placeMinimapButton(); minimapButton:Show() end
end

function ns.OnLoaded()
  registerOptions()
  ns.UpdateMinimap()
end
