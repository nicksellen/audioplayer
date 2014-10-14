function AudioPlayer(bus) {

  this.track = null;
  this.state = null;
  this.queue = [];

  var audio = document.createElement('audio');
  this.audio = audio;

  bus.subscribe('audio.now', function(track)  {
    this.track = track;
    var playableSources = track.sources.filter(function(source)  {return audio.canPlayType(source.contentType);});
    if (playableSources.length > 0) {
      var source = playableSources[0];
      audio.src = source.url;
      audio.load();
      audio.play();
      bus.send('audio.track', this.track);
    } else {
      alert('no playable sources');
    }
  }.bind(this));

  bus.subscribe('audio.request-update', function()  {
    bus.send('audio.track', this.track);
    bus.send('audio.state', this.state);
    bus.send('audio.duration', audio.duration);
    bus.send('audio.time', audio.currentTime);
  }.bind(this));

  bus.subscribe('audio.queue.push', function(track)  {
    this.queue.push(track);
  }.bind(this));

  bus.subscribe('audio.queue.clear', function()  {
    this.queue.length = 0;
  }.bind(this));

  bus.subscribe('audio.ctrl.play',  function()  {return audio.play();});
  bus.subscribe('audio.ctrl.pause', function()  {return audio.pause();});
  bus.subscribe('audio.ctrl.next',  function()  {return this.playNext();}.bind(this));

  audio.addEventListener('ended', function()  {return this.playNext();}.bind(this));

  audio.addEventListener('durationchange', function()  {
    bus.send('audio.duration', audio.duration);
  });

  audio.addEventListener('timeupdate', function()  {
    bus.send('audio.time', audio.currentTime);
  });

  audio.addEventListener('playing', function()  {
    this.state = 'playing';
    bus.send('audio.state', 'playing');
  }.bind(this));

  audio.addEventListener('pause', function()  {
    this.state = 'paused';
    bus.send('audio.state', 'paused');
  }.bind(this));

}

AudioPlayer.prototype.playNext = function() {
  if (this.queue.length > 0) {
    var next = this.queue[0];
    this.queue.splice(0, 1);
    bus.send('audio.now', next);
  }
}