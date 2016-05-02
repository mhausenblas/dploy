import os
import urllib.request, urllib.error, urllib.parse
import json
import random

from flask import Flask
app = Flask(__name__)

@app.route('/')
def sanitize():
    mesosdns_endpoint = "http://leader.mesos:8123"
    service_name = "buzzgen-dployex.marathon.mesos"
    components = service_name.split('.')
    lookup = "_" + components[0] + "._tcp." + ".".join(str(x) for x in components[1:])
    response = urllib.request.urlopen(mesosdns_endpoint + "/v1/services/" + lookup + ".")
    str_response = response.read().decode('utf-8')
    payload = json.loads(str_response)
    service_instance = random.choice(payload)
    bg_service = "http://" + service_instance['ip'] + ":" + str(service_instance['port'])
    response = urllib.request.urlopen(bg_service)
    content = response.read()
    return content*10

if __name__ == '__main__':
    app.run(port=8888)