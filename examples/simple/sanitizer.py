import os
from flask import Flask
app = Flask(__name__)

@app.route('/')
def sanitize():
    return os.environ['BGSERVER']

if __name__ == '__main__':
    app.run(port=8888)