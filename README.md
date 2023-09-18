# goffy

A CLI program to download public playlists and tracks of Spotify.

> Downloads are not done directly from Spotify, but from YouTube (if a song matches).

**goffy** does not use any official API, but its own unofficial "API" (light and rustic). Additionally, it has no features related to the private data of your Spotify account. Lastly, you can save the music on your computer or mobile device (this makes more sense).


## Features

- Download a playlist (publics only)
- Download a single track
- Download multiple tracks from a txt file

## Requirements

* **ffmpeg** (required to add metadata to audio files)

## Installation
Install by downloading [latest release](https://github.com/mathenz/goffy/releases/tag/v1.0.0).

Or executing:
```
$ go install github.com/mathenz/goffy@latest
```

## Usage

#### Download music to desktop
```
$ goffy [option] [url] -d [path/to/save/music]
```
#### Download music to mobile device
```
$ goffy [option] [url] -m
```

In case you want to download multiple tracks from a .txt file, simply change ```[url]``` to ```[path/to/txt]```.
> To correctly read all tracks from a .txt file, place the URL of each track on its own line.


### Options

```
-p   download a playlist
-t   download a single track
-f   download multiple tracks from a file.txt
```
#### On mobile devices? How does it work?

Very simple. When you set the mobile platform flag (```-m```), the music will be stored in a temporary directory on the host machine, then that folder is compressed and presented at the address ```<YOUR_HOSTMACHINE_IP>:8080```. You, from your mobile device, will access from the browser and get the music. Afterwards, both the temporary folder and the .zip file will be deleted.


### Examples

- If you want to save the music on your desktop machine:
   > 
   ```
   $ goffy -p https://open.spotify.com/playlist/37i9dQZF1EIh4XfqZs7jCB?si=5855691d6a874444 -d /path/to/musicfolder/
   ```
   ```
   $ goffy -t https://open.spotify.com/track/5WSqNyypJ0hITVpvJMetqQ?si=5d9759cc4d8d4e57 -d /path/to/musicfolder/
   ```
   ```
   $ goffy -f /path/to/file.txt -d /path/to/musicfolder/
   ```
   >
- Or if you want to save the music on your mobile device:
   > 
   ```
   $ goffy -p https://open.spotify.com/playlist/37i9dQZF1EIh4XfqZs7jCB?si=5855691d6a874444 -m
   ```
   ```
   $ goffy -t https://open.spotify.com/track/5WSqNyypJ0hITVpvJMetqQ?si=5d9759cc4d8d4e57 -m
   ```
   ```
   $ goffy -f /path/to/file.txt -m
   ```
   >


To get the URL of a playlist or a track, just click on the three dots > Share > Copy-Link-to-Playlist or Copy-Song-Link

### Contributing

Feel free to open a pull request to:

* Fix bugs
* Increase performance
* Reduce the number of lines of code (only if it does not reduce the performance)
* Suggest improvements
