 /** @jsx React.DOM */

var cx = React.addons.classSet;

function EventBus() {
  this.channels = {};
}

EventBus.prototype.subscribe = function(channel, callback) {
  var ch = this.channels[channel];
  if (!ch) {
    ch = { listeners: [] };
    this.channels[channel] = ch;
  }
  ch.listeners.push(callback);
}

EventBus.prototype.unsubscribe = function(channel, callback) {
  var ch = this.channels[channel];
  if (ch) {
    var idx = ch.listeners.indexOf(callback);
    if (idx !== -1) {
      ch.listeners.splice(idx, 1);
      if (ch.listeners.length === 0) {
        delete this.channels[channel];
      }
    }
  }
}

EventBus.prototype.send = function(channel, message) {
  setTimeout(function(){
    var ch = this.channels[channel];
    if (ch) {
      ch.listeners.forEach(function(listener){
        listener(message);
      }.bind(this));
    } else {
      console.log('unhandled mesage for', channel, ':', message);
    }
  }.bind(this), 0);
}

var bus = new EventBus();

var AlbumList = React.createClass({displayName: 'AlbumList',
  filterChanged: function(e) {
    if (this.timeout) clearTimeout(this.timeout);
    this.timeout = setTimeout(function(val){
      this.setState({ query: val });
      this.forceUpdate();
    }.bind(this, e.target.value.toLowerCase()), 200);
  },
  filter: function(album) {
    var q = this.state && this.state.query;
    if (!q) return true;
    return album.name.toLowerCase().indexOf(q) !== -1 || album.artists.toLowerCase().indexOf(q) !== -1;
  },
  shouldComponentUpdate: function(nextProps, nextState) {
    if (!nextProps || !nextProps.albums) return true;
    if (!this.albumCount) {
      this.albumCount = nextProps.albums.length;
      return true;
    } else if (this.albumCount !== nextProps.albums.length) {
      this.albumCount = nextProps.albums.length;
      return true;
    } 
    return false;
  },
  render: function(){
    return React.DOM.div({className: "album-list"}, 
      React.DOM.div({className: "search"}, 
        React.DOM.input({type: "text", placeholder: "search", onChange: this.filterChanged})
      ), 
      React.DOM.ul(null, 
        this.props.albums.filter(this.filter).map(function(album){
          var key = [album.name, album.artists].join('::');
          var url = "/albums/" + encodeURIComponent(album.name);
          return React.DOM.li({key: key}, 
            React.DOM.a({href: url}, 
              React.DOM.span({className: "artists"}, album.artists), 
              React.DOM.span({className: "name"}, album.name)
            )
            
          );
        }.bind(this))
      )
    )
  }
});

var AlbumDetail = React.createClass({displayName: 'AlbumDetail',
  play: function(track){
    bus.send('clear');
    bus.send('now', track);
    var album = this.props.album;
    var idx = album.tracks.indexOf(track);
    if (idx !== -1) {
      for (var i = idx + 1; i < album.tracks.length; i++) {
        bus.send('queue', album.tracks[i]);
      }
    }
  },
  componentDidMount: function() {
    bus.subscribe('current', function(track){
      var tracks = this.props.album.tracks;
      var updated = false;
      tracks.forEach(function(t) {
        if (track && t.id === track.id) {
          t.playing = true;
          updated = true;
        } else {
          delete t.playing;
        }
      }.bind(this));
      if (updated) {
        this.forceUpdate();
      }
    }.bind(this));
  },
  componentWillReceiveProps: function() {
    bus.send('update');
  },
  render: function(){
    var album = this.props.album;
    return React.DOM.div({className: "album-detail"}, 
      React.DOM.h2(null, album.name), 
      React.DOM.table(null, 
        React.DOM.tbody(null, 
          album.tracks.map(function(track){
            var key = track.id;
            var classes = cx({
              'playing': !!track.playing
            });
            return React.DOM.tr({key: key, className: classes, onClick: this.play.bind(this, track)}, 
              React.DOM.td({width: "40px"}, 
                React.DOM.a({className: "play-control"}, 
                  React.DOM.span({className: "icon icon-play"})
                )
              ), 
              React.DOM.td({className: "pos", width: "40px"}, track.pos), 
              React.DOM.td(null, track.name), 
              React.DOM.td(null, track.artist), 
              React.DOM.td({className: "formats", width: "80px"}, track.formats.join(' '))
            );
          }.bind(this))
        )
      )
    )
  }
});

var Track = React.createClass({displayName: 'Track',
  play: function(){
    bus.send('now', this.props.track);
  },
  render: function(){
    var track = this.props.track;
    return React.DOM.div({onClick: this.play}, track.artist, " : ", track.name);
  }
});

var AudioPlayer = React.createClass({displayName: 'AudioPlayer',
  getInitialState: function(){
    return {
      track: null,
      queue: []
    }
  },
  componentDidMount: function(){
    var audio = document.createElement('audio');
    this.audio = audio;

    bus.subscribe('now', function(track){
      this.setState({ track: track });
      var format = track.formats.indexOf('mp3') !== -1 ? 'mp3' : track.formats[0];
      var url = "/audio/" + track.id + '.' + format;
      audio.src = url;
      audio.load();
      audio.play();
      bus.send('current', this.state.track);
    }.bind(this));

    bus.subscribe('update', function(){
      bus.send('current', this.state.track);
    }.bind(this));

    bus.subscribe('queue', function(track){
      this.state.queue.push(track);
    }.bind(this));

    bus.subscribe('clear', function(){
      this.setState({ queue: [] });
    }.bind(this));

    audio.addEventListener('ended', function(){
      if (this.state.queue.length > 0) {
        var next = this.state.queue[0];
        this.state.queue.splice(0, 1);
        bus.send('now', next);
      }
    }.bind(this));

  },
  render: function(){
    var track = this.state.track;
    return React.DOM.div({className: "audio-player"}, 
      track && CurrentTrack({track: track, audio: this.audio})
    );
  }
});

var CurrentTrack = React.createClass({displayName: 'CurrentTrack',
  getInitialState: function(){
    return {
      playing: false,
      position: '--:--',
      duration: 0
    }
  },
  componentDidMount: function(){
    var audio = this.props.audio;

    audio.addEventListener('durationchange', function(){
      this.setState({ duration: audio.duration });
    }.bind(this));

    audio.addEventListener('timeupdate', function(e){
      var time = audio.currentTime;
      var minutes = Math.floor(time / 60);
      var seconds = Math.floor(time - minutes * 60);
      minutes = minutes < 10 ? '0' + minutes : '' + minutes;
      seconds = seconds < 10 ? '0' + seconds : '' + seconds;
      var progress = (audio.currentTime / this.state.duration) * 100;
      this.setState({ 
        position: minutes + ':' + seconds,
        seconds: Math.floor(audio.currentTime),
        progress: progress
      })
    }.bind(this));

    audio.addEventListener('playing', function(){
      this.setState({ playing: true });
    }.bind(this));

    audio.addEventListener('pause', function(){
      this.setState({ playing: false });
    }.bind(this));

  },
  toggle: function(){
    var audio = this.props.audio;
    if (audio.paused) {
      audio.play();
    } else {
      audio.pause();
    }
  },
  render: function(){
    var track = this.props.track;
    var playing = this.state.playing;
    var classes = cx({
      'current-track' : true,
      'playing': playing
    });
    var iconClasses = cx({
      'icon': true,
      'icon-play' : !playing,
      'icon-pause' : playing
    });
    var progressMarkerStyle = {
      left: '' + this.state.progress + '%'
    };
    return React.DOM.div({className: classes}, 
      React.DOM.div({className: "toggle", onClick: this.toggle}, 
        React.DOM.span({className: iconClasses})
      ), 
      React.DOM.div({className: "position"}, this.state.position), 
      React.DOM.div({className: "what"}, 
        React.DOM.span({className: "artist"}, track.artist), 
        React.DOM.span({className: "title"}, track.name)
      ), 
      React.DOM.div({className: "progress"}, 
        React.DOM.div({className: "marker", style: progressMarkerStyle})
      )
    );
  }
});

var MediaPlayer = React.createClass({displayName: 'MediaPlayer',
  getInitialState: function() {
    return {
      albums: []
    };
  },
  getDefaultProps: function(){
    return {
      renderPage: function(){}.bind(this)
    };
  },
  componentDidMount: function(){

    superagent.get('/api/albums', function(res) {
      this.setState({ albums: res.body.albums });
    }.bind(this));

    page('/', function(req){
      window.location = '/albums';
    }.bind(this));

    page('/albums', function(req){
      this.setProps({
        renderPage: function(){
          return React.DOM.div(null);
        }.bind(this)
      });
    }.bind(this));

    page(new RegExp("\/albums\/(.+)"), function(req){
      var name = req.params[0];
      superagent.get('/api/albums/' + encodeURIComponent(name), function(res) {
        this.setState({ album: res.body });
      }.bind(this));
      this.setProps({
        renderPage: function(){
          var album = this.state.album;
          if (album) return AlbumDetail({album: album});
        }.bind(this)
      });
    }.bind(this));

    page.start();

  },
  render: function(){
    return React.DOM.div(null, 
      AlbumList({albums: this.state.albums}), 
      this.props.renderPage(), 
      AudioPlayer(null)
    );
  }
});

React.renderComponent(MediaPlayer(null), document.getElementById('main'));