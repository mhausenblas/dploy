import os
import urllib.request, urllib.error, urllib.parse
import json
import random
from http.server import BaseHTTPRequestHandler

class BuzzConsumerHandler(BaseHTTPRequestHandler):

    def do_GET(self):
        mesosdns_endpoint = "http://leader.mesos:8123"
        service_name = "buzz-gen-dployex.marathon.mesos"
        components = service_name.split(".")
        lookup = "_" + components[0] + "._tcp." + ".".join(str(x) for x in components[1:])
        response = urllib.request.urlopen(mesosdns_endpoint + "/v1/services/" + lookup + ".")
        str_response = response.read().decode("utf-8")
        payload = json.loads(str_response)
        service_instance = random.choice(payload)
        bg_service = "http://" + service_instance["ip"] + ":" + str(service_instance["port"])
        response = urllib.request.urlopen(bg_service)
        content = response.read()
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*") # enable CORS
        self.end_headers()
        self.wfile.write(content*10)
        return 


################################################################################
# Main script
#
if __name__ == "__main__":
    from http.server import HTTPServer
    server = HTTPServer(("", 8888), BuzzConsumerHandler)
    server.serve_forever()