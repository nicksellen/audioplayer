 /** @jsx React.DOM */

var MediaPlayer = React.createClass({displayName: 'MediaPlayer',
  getInitialState: function() {
    return {
      albums: []
    };
  },
  componentDidMount: function(){
    superagent.get("/albums", function(res) {
      this.setState({ albums: res.body.albums });
    }.bind(this));
  },
  render: function(){
    var albums = this.state.albums;
    if (albums) {
    return React.DOM.div(null, albums.map(function(album){
      var key = album.name;
      return React.DOM.div({key: key}, album.name);
    }.bind(this)));
    } else {
      return React.DOM.div(null, "loading");
    }
  }
});

React.renderComponent(MediaPlayer(null), document.getElementById('main'));