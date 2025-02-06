update agp_execution
set 
    dead_at = now() + @max_heartbeat_interval,
    progress = @progress
where id = @id
and picked_by = @picked_by
returning *