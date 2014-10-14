 /** @jsx React.DOM */

var nextBusId = function(){
  var num = 0;
  return () => num++;
}();

var nextMessageId = function(){
  var num = 0;
  return () => num++;
}();

var nextSubscriberId = function(){
  var num = 0;
  return () => num++;
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
    Object.keys(this.channels).forEach(c => removeFrom(this.channels[c].listeners, identity));
  }
}

EventBus.prototype.send = function(channel, message, group) {
  setTimeout(() => {
    var ch = this.channels[channel];
    if (ch) {
      ch.listeners.forEach(listener => {
        if (!group || !listener.group || group !== listener.group) {
          listener(message, channel);
        }
      });
    }
    this.globalListeners.forEach(listener => {
      if (!group || !listener.group || group !== listener.group) {
        listener(message, channel);
      }
    });
  }, 0);
}