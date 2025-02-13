insert into agp_execution (
    created_by,
    query_id,
    query_hash,
    query,
    tier,
    secrets,
    status
) values (
    @created_by,
    @query_id,
    @query_hash,
    @query,
    @tier,
    @secrets,
    'PENDING'
)
on conflict (query_id)
where status in ('PENDING', 'RUNNING')
do update 
    set collapsed_counter = agp_execution.collapsed_counter + 1
returning *