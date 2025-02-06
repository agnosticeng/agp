insert into agp_execution (
    created_by,
    query_id,
    query_hash,
    query,
    tier,
    status
) values (
    @created_by,
    @query_id,
    @query_hash,
    @query,
    @tier,
    'PENDING'
)
on conflict (query_id)
where status in ('PENDING', 'RUNNING')
do update 
    set collapsed_counter = agp_execution.collapsed_counter + 1
returning *