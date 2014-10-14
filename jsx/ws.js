 /** @jsx React.DOM */

function WebsocketBridge() {

  this.callbacks = {};

  var protocol = location.protocol.replace('http', 'ws');
  var url = protocol + '//' + location.host + '/ws';
  this.socket = new WebSocket(url);

  this.businfos = {};
  this.runOnConnect = [];

  this.socket.addEventListener('open', () => {
    this.runOnConnect.forEach(callback => callback());
    this.registerAllBuses();
  });

  this.socket.addEventListener('message', e => {
    var data = JSON.parse(e.data);

    if (data.type === 'bus') {
      var o = this.businfos[data.busid];
      if (o) {
        o.bus.send(data.channel, data.message, 'ws');
      }
    } else if (data.type === 'connected') {
      this.registerAllBuses();
    } else if (data.type === 'register') {

      if (!data.noreply) {
        Object.keys(this.businfos).forEach(busid => {
          var businfo = this.businfos[busid];
          if (businfo.owner) {
            this.socket.send(JSON.stringify({
              type: 'register',
              busid: busid,
              name: businfo.name,
              noreply: true
            }));
          }
        });
      }

      if (!this.businfos[data.busid]) {
        var remotebus = new EventBus();
        var businfo = { busid: data.busid, name: data.name, bus: remotebus };
        this.businfos[data.busid] = businfo;
        remotebus.subscribe('*', (message, channel) => {
          this.socket.send(JSON.stringify({
            type: 'bus',
            busid: data.busid,
            message: message,
            channel: channel
          }));  
        }, 'ws');
        if (this.callbacks['remotebus']) {
          this.callbacks['remotebus'].forEach(callback => callback(businfo))
        }
      }
    }
  });

  this.socket.addEventListener('close', () => {
    console.log('ws closed');
  });

  window.addEventListener('beforeunload', () => this.socket.close());
}

WebsocketBridge.prototype.createBridgedBus = function(name, callback) {
  this.afterConnect(() => {
    var busid = guid();
    var bridgedbus = new EventBus();
    this.businfos[busid] = { name: name, owner: true, bus: bridgedbus };
    bridgedbus.subscribe('*', (message, channel) => {
      this.socket.send(JSON.stringify({
        type: 'bus',
        busid: busid,
        message: message,
        channel: channel
      }));  
    }, 'ws');
    callback(bridgedbus);
  });
}

WebsocketBridge.prototype.on = function(name, callback) {
  if (!this.callbacks[name]) {
    this.callbacks[name] = [];
  }
  this.callbacks[name].push(callback);
}

WebsocketBridge.prototype.registerAllBuses = function() {
  Object.keys(this.businfos).forEach(busid => {
    var businfo = this.businfos[busid];
    if (businfo.owner) {
      this.socket.send(JSON.stringify({
        type: 'register',
        busid: busid,
        name: businfo.name,
      }));
    }
  });
}

WebsocketBridge.prototype.afterConnect = function(callback) {
  if (this.socket.readyState === WebSocket.OPEN) {
    setTimeout(() => callback());
  } else {
    this.runOnConnect.push(callback);
  }
}