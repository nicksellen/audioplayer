 /** @jsx React.DOM */

var nextBusId = function(){
  var num = 0;
  return function()  {return num++;};
}();

var nextMessageId = function(){
  var num = 0;
  return function()  {return num++;};
}();

var nextSubscriberId = function(){
  var num = 0;
  return function()  {return num++;};
}();

function EventBus() {
  this.channels = {};
  this.globalListeners = [];
  this.id = nextBusId();
}

EventBus.prototype.subscribe = function(channel, listener, group) {
  listener.identity = nextSubscriberId();
  listener.group = group;
  if (channel === '*') {
    this.globalListeners.push(listener);
  } else {
    var ch = this.channels[channel];
    if (!ch) {
      ch = { listeners: [] };
      this.channels[channel] = ch;
    }
    ch.listeners.push(listener);
  }
  return listener.identity;
}

function removeFrom(ary, identity) {
  for (var i = 0; i < ary.length; ) {
    if (ary[i].identity === identity) {
      ary.splice(i, 1);
    } else {
      i++;
    }
  }
}

EventBus.prototype.unsubscribe = function() {
  for (var i = 0; i < arguments.length; i++) {
    var identity = arguments[i];
    removeFrom(this.globalListeners, identity);
    Object.keys(this.channels).forEach(function(c)  {return removeFrom(this.channels[c].listeners, identity);}.bind(this));
  }
}

EventBus.prototype.send = function(channel, message, group) {
  setTimeout(function()  {
    var ch = this.channels[channel];
    if (ch) {
      ch.listeners.forEach(function(listener)  {
        if (!group || !listener.group || group !== listener.group) {
          listener(message, channel);
        }
      });
    }
    this.globalListeners.forEach(function(listener)  {
      if (!group || !listener.group || group !== listener.group) {
        listener(message, channel);
      }
    });
  }.bind(this), 0);
}