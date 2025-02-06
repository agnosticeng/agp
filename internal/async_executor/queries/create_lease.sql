insert into agp_lease (key, owner, end_of_term)
values (@key, @owner, now() + @lease_duration)
on conflict (key) do update
set 
    owner = excluded.owner,
    end_of_term = excluded.end_of_term
where agp_lease.owner = excluded.owner
or agp_lease.end_of_term < now()
returning *