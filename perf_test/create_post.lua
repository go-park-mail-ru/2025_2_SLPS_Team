wrk.method = "POST"
wrk.headers["Content-Type"] = "multipart/form-data; boundary=------------------------abcd1234"

-- Функция для генерации случайного текста
function randomString(length)
    local res = ""
    local chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
    for i = 1, length do
        res = res .. string.sub(chars, math.random(1, #chars), math.random(1, #chars))
    end
    return res
end

function request()
    local boundary = "------------------------abcd1234"

    -- Формируем тело multipart/form-data
    local body = ""
    body = body .. "--" .. boundary .. "\r\n"
    body = body .. 'Content-Disposition: form-data; name="text"\r\n\r\n'
    body = body .. randomString(100) .. "\r\n"

    body = body .. "--" .. boundary .. "\r\n"
    body = body .. 'Content-Disposition: form-data; name="communityID"\r\n\r\n'
    body = body .. tostring(math.random(1, 100)) .. "\r\n"

    -- Имитация вложений
    body = body .. "--" .. boundary .. "\r\n"
    body = body .. 'Content-Disposition: form-data; name="attachments"; filename="file.txt"\r\n'
    body = body .. "Content-Type: text/plain\r\n\r\n"
    body = body .. randomString(50) .. "\r\n"

    body = body .. "--" .. boundary .. "\r\n"
    body = body .. 'Content-Disposition: form-data; name="photos"; filename="photo.jpg"\r\n'
    body = body .. "Content-Type: image/jpeg\r\n\r\n"
    body = body .. randomString(50) .. "\r\n"

    body = body .. "--" .. boundary .. "--\r\n"

    return wrk.format(nil, "/posts", nil, body)
end
