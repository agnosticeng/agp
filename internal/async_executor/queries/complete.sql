update agp_execution
set 
    status = @status,
    result = @result,
    error = @error,
    completed_at = now(),
    secrets = null
where id = @id
and picked_by = @picked_by
returning *