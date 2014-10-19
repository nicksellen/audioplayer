.width 30 30 30 3 3 4 30

select
  coalesce(albumartist, 'Various Artists') as albumartist,
  nullif(artists, albumartist) as artists,
  album,
  sum(trackcount),
  min(disc),
  min(year),
  max(toyear) as toyear,

  -- very basic metric of incompleteness
  min(mintracknumber) != 1 as incomplete 

from (

  select 

    -- we want the albumartist
    -- we might be able to generate it
    -- if all the tracks have the same artist
    coalesce(
      t.albumartist, 
      (case count(distinct t.artist)
        when 1 then t.artist
        else null -- meaning various
      end)
    ) as albumartist,

    -- but still have a nice breakdown of all of 'em
    group_concat(distinct t.artist) as artists,

    t.album, 

    count(distinct t.id) as trackcount,
    (case 
      when t.totaldiscs > 1 
        then (t.discnumber || '/' || t.totaldiscs)
      else null 
    end) as disc, -- don't want to see any 1/1's

    min(year) year,
    nullif(max(year), min(year)) toyear,

    min(t.tracknumber) as mintracknumber

  from tracks t 
  join files f on t.track_hash = f.track_hash

  where 
    t.album is not null
    and t.artist is not null
    and t.title is not null

    -- a few cases I wanted to peek at manually

    --and t.album = 'One'
    --and t.album = 'Good Rain'
    --and t.album = 'Lontano'
    --and t.album like 'Always Outnumbered%'
    --and t.album like 'Angola - The greatest%'

  group by t.album, t.albumartist

) boo

-- regroup as our inner query may have successfully
-- found an albumartist from the chaos and it might be able
-- to be paired up with it's siblings
group by album, albumartist

--order by albumartist, album
order by album

;
