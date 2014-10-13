 /** @jsx React.DOM */

var MediaPlayer = React.createClass({
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
    return <div>{albums.map(function(album){
      var key = album.name;
      return <div key={key}>{album.name}</div>;
    }.bind(this))}</div>;
    } else {
      return <div>loading</div>;
    }
  }
});

React.renderComponent(<MediaPlayer/>, document.getElementById('main'));