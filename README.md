# audioplayer

Work-in-progress audioplayer project.

The idea is to replace iTunes (+iTunes match) for me on my laptop/server/phone/web/etc and fix all known problems with music players for me, things like:

* my music collection is too big for my laptop, I'd like it to handle playing off a usb hd when it's plugged in, but when not, just play the stuff on it's internal hd, unless I have the cloud edition available, then play it from there too. basically it needs to know about the entire collection on every device and know where it can get it from, and manage a subset of the music on itself.
* I don't like alphabetical lists for anything. it means nothing to me. I *think* I'd like to position music on a 2d surface, and it's proximity to various "nodes" detirmines a strength of it's membership of that category. the categories might be unnamed but represent a loose association I have, one node might be about music I like listening to whilst eating with friends, and I can shuffle blobs around. you could grab the centre of the node and the music would follow it according to gravity-like rules (a case for d3).
* I have loads of incomplete albums, because they're really individual tracks on a playlist my mum sent me, I'd like it so that they have a "strong" connection to the playlist, and a "weak" connection to the album, this would mean by default, a list of "collections" (albums+playlists/etc) would show it under the playlist not the album.
* it has to work with MY audio files, and not depend on any cloud infrastructure (but able to use it it's around)
* it should allow remote players, so I can play it out my hifi without cables. I bought a raspberry pi and put mpd on it and it's gunna be perfect for it! (it can have a little display on it later on, and use my nice usb dac I have somewhere).
* it should cope with any part of the system going on/offline, restarting, etc

## Architecture

![architecture](http://nicksellen.co.uk/upld/audioplayer-architecture.jpg)

## Web UI

![web ui](http://nicksellen.co.uk/upld/audioplayer-webui.png)

## Prerequisites

### On Debian/Ubuntu

    apt-get install libicu52 libleveldb1 libtag1-vanilla libtagc0
