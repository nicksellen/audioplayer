 /** @jsx React.DOM */

var bus = new EventBus();
var proxy = new PlayerBusProxy(bus);
var ws = new WebsocketBridge();

ws.on('remotebus', function(businfo){
  proxy.registerBus(businfo.name, businfo.bus);
});

var remote;
if (location.pathname === '/player') {
  remote = 'remote player';
} else if (location.search) {
  remote = location.search.substring(1);
}

if (remote) {
  var name = remote;
  ws.createBridgedBus(name, function(bridgedbus)  {
    proxy.registerBus('local', new AudioPlayer(bridgedbus).bus);
  });
} else {
  proxy.registerBus('local', new AudioPlayer(new EventBus()).bus);
}

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
    return React.createElement("div", {className: "album-list"}, 
      React.createElement("div", {className: "search"}, 
        React.createElement("input", {type: "text", placeholder: "search", onChange: this.filterChanged})
      ), 
      React.createElement("ul", null, 
        this.props.albums.filter(this.filter).map(function(album)  {
          var key = [album.name, album.artists].join('::');
          var url = "/albums/" + encodeURIComponent(album.id);
          var classes = cx({
            incomplete: album.incomplete,
            complete: album.trackcount === album.totaltracks
          });
          return React.createElement("li", {key: key, className: classes}, 
            React.createElement("a", {href: url}, 
              React.createElement("span", {className: "artists"}, 
                album.albumartists, 
                album.albumartists && album.artists && ' - ', 
                album.artists
              ), 
              React.createElement("span", {className: "name"}, album.name)
            )
          );
        })
      )
    )
  }
});

var TracksView = React.createClass({displayName: 'TracksView',
  play: function(track){
    bus.send('audio.queue.clear');
    bus.send('audio.now', track);
    var tracks = this.props.tracks;
    var idx = tracks.indexOf(track);
    if (idx !== -1) {
      for (var i = idx + 1; i < tracks.length; i++) {
        bus.send('audio.queue.push', tracks[i]);
      }
    }
  },
  componentDidMount: function() {
    bus.subscribe('audio.track', function(track)  {
      var tracks = this.props.tracks;
      var updated = false;
      tracks.forEach(function(t)  {
        if (track && t.id === track.id) {
          t.playing = true;
          updated = true;
        } else {
          delete t.playing;
        }
      });
      if (updated) {
        this.forceUpdate();
      }
    }.bind(this));
  },
  componentWillReceiveProps: function() {
    bus.send('audio.request-update');
  },
  render: function(){
    return React.createElement("div", {className: "album-detail"}, 
      React.createElement("h2", null, this.props.title), 
      React.createElement("table", null, 
        React.createElement("tbody", null, 
          this.props.tracks.map(function(track)  {
            var key = track.id;
            var classes = cx({
              'playing': !!track.playing
            });
            return React.createElement("tr", {key: key, className: classes, onClick: this.play.bind(this, track)}, 
              React.createElement("td", {width: "40px"}, 
                React.createElement("a", {className: "play-control"}, 
                  React.createElement("span", {className: "icon icon-play"})
                )
              ), 
              React.createElement("td", {className: "pos", width: "40px"}, track.pos), 
              React.createElement("td", null, track.name), 
              React.createElement("td", null, track.artist), 
              React.createElement("td", {className: "formats", width: "80px"}, track.sources.map(function(source)  {return source.format;}).join(' '))
            );
          }.bind(this))
        )
      )
    )
  }
});

var Track = React.createClass({displayName: 'Track',
  play: function(){
    bus.send('audio.now', this.props.track);
  },
  render: function(){
    var track = this.props.track;
    return React.createElement("div", {onClick: this.play}, track.artist, " : ", track.name);
  }
});

var AudioPlayerSwitcher = React.createClass({displayName: 'AudioPlayerSwitcher',
  getInitialState: function(){
    return {
      players: [],
      activePlayer: null
    }
  },
  setPlayer: function(name) {
    bus.send('players.select', name);
  },
  componentDidMount: function(){
    this.sub1 = bus.subscribe('players', function(players)  {
      this.setState({ players: players });
    }.bind(this));
    this.sub2 = bus.subscribe('players.active', function(player)  {
      this.setState({ activePlayer: player });
    }.bind(this));
    bus.send('players.request-update');
  },
  componentWillUnmount: function(){
    bus.unsubscribe(this.sub1, this.sub2);
  },
  render: function(){
    return React.createElement("ul", {className: "players"}, 
      this.state.players.map(function(player)  {
        var classes = cx({
          active: this.state.activePlayer === player
        })
        return React.createElement("li", {key: player, className: classes, onClick: this.setPlayer.bind(this, player)}, player);
      }.bind(this))
    )
  }
});

var AudioControl = React.createClass({displayName: 'AudioControl',
  getInitialState: function(){
    return {
      track: null,
      playing: false,
      position: '--:--',
      duration: 0
    }
  },
  componentDidMount: function(){

    bus.subscribe('audio.track', function(track)  {
      this.setState({ track: track });
    }.bind(this));

    bus.subscribe('audio.duration', function(duration)  {
      this.setState({ duration: duration });
    }.bind(this));

    bus.subscribe('audio.time', function(time)  {
      var minutes = Math.floor(time / 60);
      var seconds = Math.floor(time - minutes * 60);
      minutes = minutes < 10 ? '0' + minutes : '' + minutes;
      seconds = seconds < 10 ? '0' + seconds : '' + seconds;
      var progress = (time / this.state.duration) * 100;
      this.setState({ 
        position: minutes + ':' + seconds,
        seconds: Math.floor(time),
        progress: progress
      })
    }.bind(this));

    bus.subscribe('audio.state', function(state)  {
      this.setState({ playing: state === 'playing' });
    }.bind(this));

  },
  toggle: function(){
    if (this.state.playing) {
      bus.send('audio.ctrl.pause');
    } else {
      bus.send('audio.ctrl.play');
    }
  },
  render: function(){
    var track = this.state.track;
    if (!track) {
      return React.createElement("div", {className: "audio-control"}, 
        React.createElement("p", null, "no track"), 
        React.createElement(AudioPlayerSwitcher, null)
      );
    }

    var playing = this.state.playing;
    var progress = this.state.progress;
    var position = this.state.position;

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
      left: '' + progress + '%'
    };
    return React.createElement("div", {className: "audio-control"}, 
      React.createElement("div", {className: classes}, 
        React.createElement("div", {className: "toggle", onClick: this.toggle}, 
          React.createElement("span", {className: iconClasses})
        ), 
        React.createElement("div", {className: "position"}, position), 
        React.createElement("div", {className: "what"}, 
          React.createElement("span", {className: "artist"}, track.artist), 
          React.createElement("span", {className: "title"}, track.name)
        ), 
        React.createElement("div", {className: "progress"}, 
          React.createElement("div", {className: "marker", style: progressMarkerStyle})
        ), 
        React.createElement(AudioPlayerSwitcher, null)
      )
    );
  }
});

var MusicBrowser = React.createClass({displayName: 'MusicBrowser',
  getInitialState: function() {
    return {
      albums: []
    };
  },
  getDefaultProps: function(){
    return {
      renderDetail: function(){}.bind(this)
    };
  },
  componentDidMount: function(){

    superagent.get('/api/albums', function(res)  {
      this.setState({ albums: res.body.albums });
    }.bind(this));

    page('/', function(req)  {
      window.location = '/albums';
    });

    page('/albums', function(req)  {
      this.setProps({
        renderDetail: function()  {return React.createElement("div", null);}
      });
    }.bind(this));

    page(new RegExp("\/albums\/(.+)"), function(req)  {
      var name = req.params[0];
      superagent.get('/api/albums/' + encodeURIComponent(name), function(res)  {
        var album = res.body;
        this.setProps({
          renderDetail: function()  {return React.createElement(TracksView, {title: album.name, tracks: album.tracks});}
        });
      }.bind(this));
    }.bind(this));

    page.start();

  },
  render: function(){
    return React.createElement("div", null, 
      React.createElement(AlbumList, {albums: this.state.albums}), 
      this.props.renderDetail(), 
      React.createElement(AudioControl, null)
    );
  }
});

var AudioStatus = React.createClass({displayName: 'AudioStatus',
  getInitialState: function(){
    return {
      track: null,
      playing: false,
      position: '--:--',
      duration: 0
    }
  },
  componentDidMount: function(){

    bus.subscribe('audio.track', function(track)  {
      this.setState({ track: track });
      document.title = track.artist + ' - ' + track.name;
    }.bind(this));

    bus.subscribe('audio.duration', function(duration)  {
      this.setState({ duration: duration });
    }.bind(this));

    bus.subscribe('audio.time', function(time)  {
      var minutes = Math.floor(time / 60);
      var seconds = Math.floor(time - minutes * 60);
      minutes = minutes < 10 ? '0' + minutes : '' + minutes;
      seconds = seconds < 10 ? '0' + seconds : '' + seconds;
      var progress = (time / this.state.duration) * 100;
      this.setState({ 
        position: minutes + ':' + seconds,
        seconds: Math.floor(time),
        progress: progress
      })
    }.bind(this));

    bus.subscribe('audio.state', function(state)  {
      this.setState({ playing: state === 'playing' });
    }.bind(this));

  },
  render: function(){
    var track = this.state.track;
    if (!track) {
      return React.createElement("div", {className: "audio-status"}, 
        React.createElement("p", null, "no track")
      );
    }

    var playing = this.state.playing;
    var progress = this.state.progress;
    var position = this.state.position;

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
      left: '' + progress + '%'
    };
    return React.createElement("div", {className: "audio-status"}, 
      React.createElement("div", {className: classes}, 
        React.createElement("div", {className: "position"}, position), 
        React.createElement("div", {className: "what"}, 
          React.createElement("span", {className: "artist"}, track.artist), 
          React.createElement("span", {className: "title"}, track.name)
        ), 
        React.createElement("div", {className: "progress"}, 
          React.createElement("div", {className: "marker", style: progressMarkerStyle})
        )
      )
    );
  }
});

var RemotePlayer = React.createClass({displayName: 'RemotePlayer',
  render: function(){
    return React.createElement(AudioStatus, null);
  }
});

if (location.pathname === '/player') {
  document.querySelector('body').className = 'full';
  React.renderComponent(React.createElement(RemotePlayer, null), document.getElementById('main'));
} else {
  React.renderComponent(React.createElement(MusicBrowser, null), document.getElementById('main'));
}