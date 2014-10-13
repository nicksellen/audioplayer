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

var AlbumList = React.createClass({
  filterChanged: function(e) {
    this.setState({ query: e.target.value.toLowerCase() });
  },
  filter: function(album) {
    var q = this.state && this.state.query;
    if (!q) return true;
    return album.name.toLowerCase().indexOf(q) !== -1 || album.artists.toLowerCase().indexOf(q) !== -1;
  },
  render: function(){
    return <div className="album-list">
      <div className="search">
        <input type="text" placeholder="search" onChange={this.filterChanged}/>
      </div>
      <ul>
        {this.props.albums.filter(this.filter).map(function(album){
          var key = [album.name, album.artists].join('::');
          var url = "/albums/" + encodeURIComponent(album.name);
          return <li key={key}>
            <a href={url}>
              <span className="artists">{album.artists}</span>
              <span className="name">{album.name}</span>
            </a>
            
          </li>;
        }.bind(this))}
      </ul>
    </div>
  }
});

var AlbumDetail = React.createClass({
  play: function(track){
    bus.send('track', track);
  },
  render: function(){
    var album = this.props.album;
    return <div className="album-detail">
      <h2>{album.name}</h2>
      <table>
        <tbody>
          {album.tracks.map(function(track){
            var key = track.id;
            return <tr key={key} onClick={this.play.bind(this, track)}>
              <td width="40px">
                <a className="play-control">
                  <span className="icon icon-play"></span>
                </a>
              </td>
              <td width="40px">{track.pos}</td>
              <td>{track.name}</td>
              <td>{track.artist}</td>
              <td className="formats" width="80px">{track.formats.join(' ')}</td>
            </tr>;
          }.bind(this))}
        </tbody>
      </table>
    </div>
  }
});

var Track = React.createClass({
  play: function(){
    bus.send('track', this.props.track);
  },
  render: function(){
    var track = this.props.track;
    return <div onClick={this.play}>{track.artist} : {track.name}</div>;
  }
});

var AudioPlayer = React.createClass({
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

    page('/', function(req){
      window.location = '/albums';
    }.bind(this));

    page('/albums', function(req){
      this.setProps({
        renderPage: function(){
          return <div/>;
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
          if (album) return <AlbumDetail album={album}/>;
        }.bind(this)
      });
    }.bind(this));

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