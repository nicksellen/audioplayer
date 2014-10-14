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
  ws.createBridgedBus(name, bridgedbus => {
    proxy.registerBus('local', new AudioPlayer(bridgedbus).bus);
  });
} else {
  proxy.registerBus('local', new AudioPlayer(new EventBus()).bus);
}

var AlbumList = React.createClass({
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
    return <div className="album-list">
      <div className="search">
        <input type="text" placeholder="search" onChange={this.filterChanged}/>
      </div>
      <ul>
        {this.props.albums.filter(this.filter).map(album => {
          var key = [album.name, album.artists].join('::');
          var url = "/albums/" + encodeURIComponent(album.name);
          return <li key={key}>
            <a href={url}>
              <span className="artists">{album.artists}</span>
              <span className="name">{album.name}</span>
            </a>
          </li>;
        })}
      </ul>
    </div>
  }
});

var TracksView = React.createClass({
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
    bus.subscribe('audio.track', track => {
      var tracks = this.props.tracks;
      var updated = false;
      tracks.forEach(t => {
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
    });
  },
  componentWillReceiveProps: function() {
    bus.send('audio.request-update');
  },
  render: function(){
    return <div className="album-detail">
      <h2>{this.props.title}</h2>
      <table>
        <tbody>
          {this.props.tracks.map(track => {
            var key = track.id;
            var classes = cx({
              'playing': !!track.playing
            });
            return <tr key={key} className={classes} onClick={this.play.bind(this, track)}>
              <td width="40px">
                <a className="play-control">
                  <span className="icon icon-play"></span>
                </a>
              </td>
              <td className="pos" width="40px">{track.pos}</td>
              <td>{track.name}</td>
              <td>{track.artist}</td>
              <td className="formats" width="80px">{track.sources.map(source => source.format).join(' ')}</td>
            </tr>;
          })}
        </tbody>
      </table>
    </div>
  }
});

var Track = React.createClass({
  play: function(){
    bus.send('audio.now', this.props.track);
  },
  render: function(){
    var track = this.props.track;
    return <div onClick={this.play}>{track.artist} : {track.name}</div>;
  }
});

var AudioPlayerSwitcher = React.createClass({
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
    this.sub1 = bus.subscribe('players', players => {
      this.setState({ players: players });
    });
    this.sub2 = bus.subscribe('players.active', player => {
      this.setState({ activePlayer: player });
    });
    bus.send('players.request-update');
  },
  componentWillUnmount: function(){
    bus.unsubscribe(this.sub1, this.sub2);
  },
  render: function(){
    return <ul className="players">
      {this.state.players.map(player => {
        var classes = cx({
          active: this.state.activePlayer === player
        })
        return <li key={player} className={classes} onClick={this.setPlayer.bind(this, player)}>{player}</li>;
      })}
    </ul>
  }
});

var AudioControl = React.createClass({
  getInitialState: function(){
    return {
      track: null,
      playing: false,
      position: '--:--',
      duration: 0
    }
  },
  componentDidMount: function(){

    bus.subscribe('audio.track', track => {
      this.setState({ track: track });
    });

    bus.subscribe('audio.duration', duration => {
      this.setState({ duration: duration });
    });

    bus.subscribe('audio.time', time => {
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
    });

    bus.subscribe('audio.state', state => {
      this.setState({ playing: state === 'playing' });
    });

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
      return <div className="audio-control">
        <p>no track</p>
        <AudioPlayerSwitcher/>
      </div>;
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
    return <div className="audio-control">
      <div className={classes}>
        <div className="toggle" onClick={this.toggle}>
          <span className={iconClasses}></span>
        </div>
        <div className="position">{position}</div>
        <div className="what">
          <span className="artist">{track.artist}</span>
          <span className="title">{track.name}</span>
        </div>
        <div className="progress">
          <div className="marker" style={progressMarkerStyle}></div>
        </div>
        <AudioPlayerSwitcher/>
      </div>
    </div>;
  }
});

var MusicBrowser = React.createClass({
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

    superagent.get('/api/albums', function(res) {
      this.setState({ albums: res.body.albums });
    }.bind(this));

    page('/', req => {
      window.location = '/albums';
    });

    page('/albums', req => {
      this.setProps({
        renderDetail: () => <div/>
      });
    });

    page(new RegExp("\/albums\/(.+)"), req => {
      var name = req.params[0];
      superagent.get('/api/albums/' + encodeURIComponent(name), res => {
        var album = res.body;
        this.setProps({
          renderDetail: () => {
            if (album) return <TracksView title={album.name} tracks={album.tracks}/>;
          }
        });
      });
    });

    page.start();

  },
  render: function(){
    return <div>
      <AlbumList albums={this.state.albums}/>
      {this.props.renderDetail()}
      <AudioControl/>
    </div>;
  }
});

var AudioStatus = React.createClass({
  getInitialState: function(){
    return {
      track: null,
      playing: false,
      position: '--:--',
      duration: 0
    }
  },
  componentDidMount: function(){

    bus.subscribe('audio.track', track => {
      this.setState({ track: track });
    });

    bus.subscribe('audio.duration', duration => {
      this.setState({ duration: duration });
    });

    bus.subscribe('audio.time', time => {
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
    });

    bus.subscribe('audio.state', state => {
      this.setState({ playing: state === 'playing' });
    });

  },
  render: function(){
    var track = this.state.track;
    if (!track) {
      return <div className="audio-status">
        <p>no track</p>
      </div>;
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
    return <div className="audio-status">
      <div className={classes}>
        <div className="position">{position}</div>
        <div className="what">
          <span className="artist">{track.artist}</span>
          <span className="title">{track.name}</span>
        </div>
        <div className="progress">
          <div className="marker" style={progressMarkerStyle}></div>
        </div>
      </div>
    </div>;
  }
});

var RemotePlayer = React.createClass({
  render: function(){
    return <AudioStatus/>;
  }
});

if (location.pathname === '/player') {
  document.querySelector('body').className = 'full';
  React.renderComponent(<RemotePlayer/>, document.getElementById('main'));
} else {
  React.renderComponent(<MusicBrowser/>, document.getElementById('main'));
}