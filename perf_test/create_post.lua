wrk.method = "POST"

local session_id = os.getenv("SESSION_ID")
local csrf_token = os.getenv("CSRF_TOKEN")

if not session_id or not csrf_token then
  error("SESSION_ID or CSRF_TOKEN not provided")
end

local function random_string(len)
  local chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
  local res = {}
  for i = 1, len do
    res[i] = chars:sub(math.random(#chars), math.random(#chars))
  end
  return table.concat(res)
end

local function build_body()
  local boundary = "----wrkBoundary" .. random_string(12)

  local body = ""
  body = body .. "--" .. boundary .. "\r\n"
  body = body .. 'Content-Disposition: form-data; name="text"\r\n\r\n'
  body = body .. "Perf test post " .. random_string(50) .. "\r\n"


  body = body .. "--" .. boundary .. "\r\n"
  body = body .. 'Content-Disposition: form-data; name="attachments"; filename="file.txt"\r\n'
  body = body .. "Content-Type: text/plain\r\n\r\n"
  body = body .. "Test file content " .. random_string(100) .. "\r\n"

  body = body .. "--" .. boundary .. "--\r\n"

  return body, boundary
end

request = function()
  local body, boundary = build_body()

  local headers = {
    ["Content-Type"] = "multipart/form-data; boundary=" .. boundary,
    ["Cookie"] = "session_id=" .. session_id .. "; CSRF_token=" .. csrf_token,
    ["X-CSRF-Token"] = csrf_token,
    ["Accept"] = "application/json"
  }

  return wrk.format("POST", "/api/posts", headers, body)
end
