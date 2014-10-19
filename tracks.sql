.width 30 30 3 30

select 
  coalesce(t.albumartist, t.artist), 
  t.album, 
  t.discnumber || '/' || t.totaldiscs,
  t.tracknumber || '/' || t.totaltracks,
  t.title
from tracks t 
join files f on t.track_hash = f.track_hash

where 
  t.album is not null and 
  t.artist is not null
group by t.track_hash
order by t.album, t.discnumber, coalesce(t.albumartist, t.artist), t.tracknumber
limit 1000

;
