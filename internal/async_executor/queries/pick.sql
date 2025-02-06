update agp_execution
set 
	status = 'RUNNING',
	picked_at = now(),
	picked_by = @picked_by,
	dead_at = now() + @max_heartbeat_interval
where id = (
	select 
		id
	from agp_execution
	where tier = @tier
	and status = 'PENDING'
	order by created_at asc
	for update skip locked
	limit 1
)
returning *