import requests

{
    "song_id":"themessage",
    "difficulty":2,
    "rating":11.20042,
    "score":9900084,
    "shiny_perfect_count":891,
    "perfect_count":977,
    "near_count":10,
    "miss_count":5,
    "health":100,
    "time_played":1597208335,
    "modifier":0,
    "clear_type":1
}

url = 'http://192.168.124.2:8080/coffee/12/score/song'
auth = 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoxLCJleHAiOjE1OTc5MTI5NjUsImlzcyI6IlpyY2FlYSJ9.rUVXvsGgx7iOGoPOj3UMXvOFx8ASG6OhvhKAjqPwJ2U'
headers = {'Authorization': auth}
data = '?song_id=themessage&difficulty=2&score=9900084&shiny_perfect_count=891&perfect_count=977&near_count=10&miss_count=5&health=100&time_played=1597208335&modifier=0&clear_type=1&modifier=0&beyond_gauge=0&health=100'
resp = requests.post(url+data, headers=headers)
resp.raise_for_status()
print(resp.content)
