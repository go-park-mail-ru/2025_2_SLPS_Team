wrk.method = "GET"

local session_id = os.getenv("SESSION_ID")
local csrf_token = os.getenv("CSRF_TOKEN")

if not session_id then
  error("SESSION_ID not provided")
end

local function rand(min, max)
  return math.random(min, max)
end

request = function()
  local page = rand(1, 100000)
  local limit = rand(10, 20)

  local path = string.format(
    "/api/posts?page=%d&limit=%d",
    page,
    limit
  )

  local headers = {
    ["Accept"] = "application/json",
    ["Cookie"] = "session_id=" .. session_id .. "; CSRF_token=" .. csrf_token
  }

  return wrk.format("GET", path, headers)
end
