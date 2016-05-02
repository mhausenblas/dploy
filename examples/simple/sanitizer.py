import os
import urllib2
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
    print lookup
    payload = json.load(urllib2.urlopen(mesosdns_endpoint + "/v1/services/" + lookup + "."))
    print payload
    service_instance = random.choice(payload)
    bg_service = "http://" + service_instance['ip'] + ":" + str(service_instance['port'])
    print bg_service
    response = urllib2.urlopen(bg_service)
    content = response.read()
    return content*10

if __name__ == '__main__':
    app.run(port=8888)