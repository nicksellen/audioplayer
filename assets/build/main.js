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
  var ch = this.channels[channel];
  if (ch) {
    ch.listeners.forEach(function(listener){
      listener(message);
    });
  }
}

var bus = new EventBus();

var AlbumList = React.createClass({displayName: 'AlbumList',
  filterChanged: function(e) {
    this.setState({ query: e.target.value.toLowerCase() });
  },
  filter: function(album) {
    var q = this.state && this.state.query;
    if (!q) return true;
    return album.name.toLowerCase().indexOf(q) !== -1 || album.artists.toLowerCase().indexOf(q) !== -1;
  },
  render: function(){
    return React.DOM.div({className: "album-list"}, 
      React.DOM.div({className: "search"}, 
        React.DOM.input({type: "text", placeholder: "search", onChange: this.filterChanged})
      ), 
      React.DOM.ul(null, 
        this.props.albums.filter(this.filter).map(function(album){
          var key = album.name;
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
    bus.send('track', track);
  },
  render: function(){
    var album = this.props.album;
    return React.DOM.div({className: "album-detail"}, 
      React.DOM.h2(null, album.name), 
      React.DOM.table(null, 
        React.DOM.tbody(null, 
          album.tracks.map(function(track){
            var key = track.id;
            return React.DOM.tr({key: key, onClick: this.play.bind(this, track)}, 
              React.DOM.td({width: "40px"}, 
                React.DOM.a({className: "play-control"}, 
                  React.DOM.span({className: "icon icon-play"})
                )
              ), 
              React.DOM.td({width: "40px"}, track.pos), 
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
    bus.send('track', this.props.track);
  },
  render: function(){
    var track = this.props.track;
    return React.DOM.div({onClick: this.play}, track.artist, " : ", track.name);
  }
});

var AudioPlayer = React.createClass({displayName: 'AudioPlayer',
  getInitialState: function(){
    return {
      track: null
    }
  },
  componentDidMount: function(){
    var audio = document.createElement('audio');
    this.audio = audio;
    bus.subscribe('track', function(track){
      this.setState({ track: track });
      var format = track.formats.indexOf('mp3') !== -1 ? 'mp3' : track.formats[0];
      var url = "/audio/" + track.id + '.' + format;
      audio.src = url;
      audio.load();
      audio.play();
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
          return AlbumList({albums: this.state.albums});
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
          return React.DOM.div(null, 
            AlbumList({albums: this.state.albums}), 
            album && AlbumDetail({album: album})
          );
        }.bind(this)
      });
    }.bind(this));

    page.start();
  },
  render: function(){
    return React.DOM.div(null, 
      this.props.renderPage(), 
      AudioPlayer(null)
    );
  }
});

React.renderComponent(MediaPlayer(null), document.getElementById('main'));