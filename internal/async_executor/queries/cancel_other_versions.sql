update agp_execution
set 
    status = 'CANCELED'
where query_id = @query_id
and query_hash <> @query_hash
and status in ('PENDING', 'RUNNING')