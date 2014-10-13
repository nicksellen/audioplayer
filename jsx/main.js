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
      ch.listeners.forEach(listener => {
        listener(message);
      });
    } else {
      console.log('unhandled mesage for', channel, ':', message);
    }
  }.bind(this), 0);
}

var bus = new EventBus();

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

var AlbumDetail = React.createClass({
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
    bus.subscribe('current', track => {
      var tracks = this.props.album.tracks;
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
    bus.send('update');
  },
  render: function(){
    var album = this.props.album;
    return <div className="album-detail">
      <h2>{album.name}</h2>
      <table>
        <tbody>
          {album.tracks.map(track => {
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
              <td className="formats" width="80px">{track.formats.join(' ')}</td>
            </tr>;
          })}
        </tbody>
      </table>
    </div>
  }
});

var Track = React.createClass({
  play: function(){
    bus.send('now', this.props.track);
  },
  render: function(){
    var track = this.props.track;
    return <div onClick={this.play}>{track.artist} : {track.name}</div>;
  }
});

var AudioPlayer = React.createClass({
  getInitialState: function(){
    return {
      track: null,
      queue: []
    }
  },
  componentDidMount: function(){
    var audio = document.createElement('audio');
    this.audio = audio;

    bus.subscribe('now', track => {
      this.setState({ track: track });
      var format = track.formats.indexOf('mp3') !== -1 ? 'mp3' : track.formats[0];
      var url = "/audio/" + track.id + '.' + format;
      audio.src = url;
      audio.load();
      audio.play();
      bus.send('current', this.state.track);
    });

    bus.subscribe('update', () => {
      bus.send('current', this.state.track);
    });

    bus.subscribe('queue', track => {
      this.state.queue.push(track);
    });

    bus.subscribe('clear', () => {
      this.setState({ queue: [] });
    });

    audio.addEventListener('ended', () => {
      if (this.state.queue.length > 0) {
        var next = this.state.queue[0];
        this.state.queue.splice(0, 1);
        bus.send('now', next);
      }
    });

  },
  render: function(){
    var track = this.state.track;
    return <div className="audio-player">
      {track && <CurrentTrack track={track} audio={this.audio}/>}
    </div>;
  }
});

var CurrentTrack = React.createClass({
  getInitialState: function(){
    return {
      playing: false,
      position: '--:--',
      duration: 0
    }
  },
  componentDidMount: function(){
    var audio = this.props.audio;

    audio.addEventListener('durationchange', () => {
      this.setState({ duration: audio.duration });
    });

    audio.addEventListener('timeupdate', e => {
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
    });

    audio.addEventListener('playing', () => {
      this.setState({ playing: true });
    });

    audio.addEventListener('pause', () => {
      this.setState({ playing: false });
    });

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
    return <div className={classes}>
      <div className="toggle" onClick={this.toggle}>
        <span className={iconClasses}></span>
      </div>
      <div className="position">{this.state.position}</div>
      <div className="what">
        <span className="artist">{track.artist}</span>
        <span className="title">{track.name}</span>
      </div>
      <div className="progress">
        <div className="marker" style={progressMarkerStyle}></div>
      </div>
    </div>;
  }
});

var MediaPlayer = React.createClass({
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

    page('/', req => {
      window.location = '/albums';
    });

    page('/albums', req => {
      this.setProps({
        renderPage: () => <div/>;
      });
    });

    page(new RegExp("\/albums\/(.+)"), req => {
      var name = req.params[0];
      superagent.get('/api/albums/' + encodeURIComponent(name), function(res) {
        this.setState({ album: res.body });
      }.bind(this));
      this.setProps({
        renderPage: () => {
          var album = this.state.album;
          if (album) return <AlbumDetail album={album}/>;
        }
      });
    });

    page.start();

  },
  render: function(){
    return <div>
      <AlbumList albums={this.state.albums}/>
      {this.props.renderPage()}
      <AudioPlayer/>
    </div>;
  }
});

React.renderComponent(<MediaPlayer/>, document.getElementById('main'));