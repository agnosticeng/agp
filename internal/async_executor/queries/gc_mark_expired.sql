update agp_execution
set
    status = 'EXPIRED'
where id in (
    select 
        id
    from agp_execution
    where (status = 'CANCELED'  and (age(now(), created_at)   > @canceled_expiration))
    or    (status = 'FAILED'    and (age(now(), completed_at) > @failed_expiration))
    or    (status = 'SUCCEEDED' and (age(now(), completed_at) > @succeeded_expiration))
    limit @limit
)
returning id