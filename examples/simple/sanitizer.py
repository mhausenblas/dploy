import os
import urllib2

from flask import Flask
app = Flask(__name__)

@app.route('/')

def lookup_service(service_name):
    mesosdns_endpoint = "leader.mesos:8123"
    components = service_name.split('.')
    lookup = "_" + components[0] + "._tcp." + ".".join(str(x) for x in components[1:])
    payload = json.load(urllib2.urlopen(mesosdns_endpoint + "/v1/services/" + lookup + "."))
    service_instance = random.choice(payload)
    return (service_instance['ip'], service_instance['port'])

def sanitize():
    ip, port = lookup_service("buzzgen-dployex.marathon.mesos")
    response = urllib2.urlopen("%s:%s" %(ip, str(port)))
    content = response.read()
    return content*10

if __name__ == '__main__':
    app.run(port=8888)