 /** @jsx React.DOM */

var PROXY_GROUP = 'master';

function PlayerBusProxy(masterBus) {

  this.players = {};
  this.masterBus = masterBus;
  this.current = null;

  // proxy messages from master -> active 

  this.masterBus.subscribe('*', function(message, channel)  {
    if (!this.current) return;
    this.current.bus.send(channel, message, PROXY_GROUP);
  }.bind(this), PROXY_GROUP);

  this.masterBus.subscribe('players.request-update', function()  {
    if (this.current) { 
      this.masterBus.send('players.active', this.current.name, PROXY_GROUP);
      this.masterBus.send('players', Object.keys(this.players), PROXY_GROUP);
    }
  }.bind(this), PROXY_GROUP);

  this.masterBus.subscribe('players.select', function(name)  {
    var o = this.players[name];
    if (o && (o !== this.current)) {
      this.current = o;
      this.current.bus.send('audio.request-update');
      this.masterBus.send('players.active', this.current.name);
    } else {
      console.log('couldn\'t find player for', name);
    }
  }.bind(this));

}

PlayerBusProxy.prototype.registerBus = function(name, bus) {
  var o = {};
  o.name = name;
  o.bus = bus;
  this.players[name] = o;

  // proxy messages from this bus -> master

  o.bus.subscribe('*', function(message, channel)  {
    if (this.current && (o.bus === this.current.bus)) {
      this.masterBus.send(channel, message, PROXY_GROUP);
    }
  }.bind(this), PROXY_GROUP);

  this.masterBus.send('players', Object.keys(this.players), PROXY_GROUP);

  if (Object.keys(this.players).length === 1) {
    this.masterBus.send('players.select', name);
  }

};