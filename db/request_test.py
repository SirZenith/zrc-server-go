import requests

{
    "song_id":"",
    "difficulty":2,
    "rating":0,
    "score":10000000,
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
auth = 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoxLCJleHAiOjE1OTk2NzYwMDgsImlzcyI6IlpyY2FlYSJ9.a1xs2Ee3B7BD9-Tl0Vb4D_cb3QT2U90rHbq-zIZDAhk'
headers = {'Authorization': auth}
data = '?song_id=themessage&difficulty=2&score=0&shiny_perfect_count=891&perfect_count=977&near_count=10&miss_count=5&health=100&time_played=1597208335&modifier=0&clear_type=1&modifier=0&beyond_gauge=0&health=100'
resp = requests.post(url+data, headers=headers)
resp.raise_for_status()
print(resp.content)
