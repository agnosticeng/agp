delete from agp_lease 
where key = @key
and owner = @owner
and end_of_term > now()
returning *
