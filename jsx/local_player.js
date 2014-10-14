 /** @jsx React.DOM */

function AudioPlayer(bus) {

  this.bus = bus;

  this.track = null;
  this.state = null;
  this.queue = [];

  var audio = document.createElement('audio');
  this.audio = audio;

  bus.subscribe('audio.now', track => {
    this.track = track;
    var playableSources = track.sources.filter(source => audio.canPlayType(source.contentType));
    if (playableSources.length > 0) {
      var source = playableSources[0];
      audio.src = source.url;
      audio.load();
      audio.play();
      bus.send('audio.track', this.track);
    } else {
      alert('no playable sources');
    }
  });

  bus.subscribe('audio.request-update', () => {
    bus.send('audio.track', this.track);
    bus.send('audio.state', this.state);
    bus.send('audio.duration', audio.duration);
    bus.send('audio.time', audio.currentTime);
  });

  bus.subscribe('audio.queue.push', track => {
    this.queue.push(track);
  });

  bus.subscribe('audio.queue.clear', () => {
    this.queue.length = 0;
  });

  bus.subscribe('audio.ctrl.play',  () => audio.play());
  bus.subscribe('audio.ctrl.pause', () => audio.pause());
  bus.subscribe('audio.ctrl.next',  () => this.playNext());

  audio.addEventListener('ended', () => this.playNext());

  audio.addEventListener('durationchange', () => {
    bus.send('audio.duration', audio.duration);
  });

  audio.addEventListener('timeupdate', () => {
    bus.send('audio.time', audio.currentTime);
  });

  audio.addEventListener('playing', () => {
    this.state = 'playing';
    bus.send('audio.state', 'playing');
  });

  audio.addEventListener('pause', () => {
    this.state = 'paused';
    bus.send('audio.state', 'paused');
  });

}

AudioPlayer.prototype.playNext = function() {
  if (this.queue.length > 0) {
    var next = this.queue[0];
    this.queue.splice(0, 1);
    bus.send('audio.now', next);
  }
}