-- Logbook Lua frontend scripts
-- Runs in browser via Fengari (Lua VM compiled to JS)

local doc = js.global.document
local win = js.global

local themes = {"light", "nord", "system"}
local themeLabels = {light = "☀ Light", nord = "☾ Dark", system = "◐ Auto"}
local themeTitles = {light = "Light mode", nord = "Dark (Nord) mode", system = "Follow system preference"}

-- Theme toggle
function setupThemeToggle()
    local btn = doc:getElementById("theme-toggle")
    if not btn then return end

    local saved = win.localStorage:getItem("logbook-theme")
    if saved then
        applyTheme(saved)
    else
        local server = doc.documentElement:getAttribute("data-theme")
        if server == "" then server = "light" end
        applyTheme(server)
    end

    btn:addEventListener("click", function()
        local cur = doc.documentElement:getAttribute("data-theme")
        if cur == "" then cur = "light" end
        local idx = 0
        for i, t in ipairs(themes) do
            if t == cur then idx = i; break end
        end
        local next = themes[(idx % #themes) + 1]
        applyTheme(next)
    end)
end

local function isDark()
    local ok, mq = pcall(function() return win:matchMedia("(prefers-color-scheme: dark)") end)
    if ok and mq then
        local ok2, val = pcall(function() return mq:get("matches") end)
        if ok2 then return val end
    end
    return false
end

local function setFont(theme)
    local linkMono = doc:getElementById("font-link")
    local linkSans = doc:getElementById("font-link-inter")
    if not linkMono or not linkSans then return end
    local useMono = theme == "nord" or (theme == "system" and isDark())
    if useMono then
        linkMono.href = "https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:ital,wght@0,400;0,500;0,600;0,700;1,400&display=swap"
        linkSans.href = "data:text/css,/* no font */"
    else
        linkMono.href = "data:text/css,/* no font */"
        linkSans.href = "https://fonts.googleapis.com/css2?family=Inter:ital,wght@0,400;0,500;0,600;0,700;1,400&display=swap"
    end
end

function applyTheme(theme)
    if not themeLabels[theme] then theme = "light" end
    doc.documentElement:setAttribute("data-theme", theme)
    win.localStorage:setItem("logbook-theme", theme)
    win:eval('document.cookie="logbook-theme=' .. theme .. '; path=/; max-age=' .. tostring(86400 * 365) .. '"')
    setFont(theme)
    local btn = doc:getElementById("theme-toggle")
    if btn then
        btn.textContent = themeLabels[theme]
        btn.title = themeTitles[theme]
    end
end

-- Listen for OS theme changes when in "system" mode
local function watchSystemTheme()
    local mq = win:matchMedia("(prefers-color-scheme: dark)")
    mq:addEventListener("change", function()
        local cur = doc.documentElement:getAttribute("data-theme")
        if cur == "system" then
            setFont("system")
        end
    end)
end

-- Confirm dialog for delete forms
function setupConfirmDialogs()
    local forms = doc:querySelectorAll('form[data-lua="confirm-delete"]')
    for i = 0, forms.length - 1 do
        forms[i]:addEventListener("submit", function(e)
            e:preventDefault()
            showConfirm("Are you sure you want to delete this?", function()
                forms[i]:submit()
            end)
        end)
    end
end

function showConfirm(message, callback)
    local overlay = doc:createElement("div")
    overlay.className = "confirm-overlay"

    local dialog = doc:createElement("div")
    dialog.className = "confirm-dialog"

    local p = doc:createElement("p")
    p.textContent = message

    local actions = doc:createElement("div")
    actions.className = "form-actions"

    local yesBtn = doc:createElement("button")
    yesBtn.textContent = "Delete"
    yesBtn.className = "btn btn-danger"
    yesBtn:addEventListener("click", function()
        doc.body:removeChild(overlay)
        callback()
    end)

    local noBtn = doc:createElement("button")
    noBtn.textContent = "Cancel"
    noBtn.className = "btn btn-secondary"
    noBtn:addEventListener("click", function()
        doc.body:removeChild(overlay)
    end)

    actions:appendChild(yesBtn)
    actions:appendChild(noBtn)
    dialog:appendChild(p)
    dialog:appendChild(actions)
    overlay:appendChild(dialog)
    doc.body:appendChild(overlay)
end

-- Validate flight form
function setupFlightFormValidation()
    local forms = doc:querySelectorAll('form[data-lua="flight-form"]')
    for i = 0, forms.length - 1 do
        forms[i]:addEventListener("submit", function(e)
            local total = forms[i]:querySelector('input[name="totalTime"]')
            if total and total.value ~= "" then
                local val = tonumber(total.value)
                if val and val <= 0 then
                    e:preventDefault()
                    win:alert("Total time must be greater than 0")
                    total:focus()
                end
            end
        end)
    end
end

-- Auto-submit filter form on text input (debounced); select/date require manual Apply
function setupAutoFilter()
    local form = doc:querySelector('form[data-lua="filter-form"]')
    if not form then return end

    local inputs = form:querySelectorAll("input[type='text']")
    local timer = nil

    for i = 0, inputs.length - 1 do
        inputs[i]:addEventListener("input", function()
            if timer then clearTimeout(timer) end
            timer = win:setTimeout(function()
                form:submit()
                timer = nil
            end, 500)
        end)
    end
end

-- Set default date on flight form
function setDefaultDate()
    local dateInput = doc:querySelector('input[name="date"]')
    if dateInput and dateInput.value == "" then
        local d = js.global.Date:new()
        local month = tostring(d:getMonth() + 1)
        local day = tostring(d:getDate())
        local year = tostring(d:getFullYear())
        if #month == 1 then month = "0" .. month end
        if #day == 1 then day = "0" .. day end
        dateInput.value = year .. "-" .. month .. "-" .. day
    end
end

-- Init
(function()
    setupThemeToggle()
    watchSystemTheme()
    setupConfirmDialogs()
    setupFlightFormValidation()
    setupAutoFilter()
    setDefaultDate()
end)()
