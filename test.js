import fs from "fs";

import WebSocket from "ws";

let url;
url = 'wss://192.168.99.101:8443/api/v1/namespaces/default/pods/example/exec?command=echo&command=foo&stderr=true&stdout=true';

let sock;
sock = new WebSocket(url, {
    ca: fs.readFileSync(`${process.env.HOME}/.minikube/ca.crt`),
    cert: fs.readFileSync(`${process.env.HOME}/.minikube/client.crt`),
    key: fs.readFileSync(`${process.env.HOME}/.minikube/client.key`)
});

sock.on('upgrade', x => console.log('upgrade', x.headers.upgrade));
sock.on('open', () => console.log('open'));
sock.on('message', x => console.log('message', JSON.stringify(x.toString())));
sock.on('close', () => console.log('close'));
