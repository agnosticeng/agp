update agp_execution
set
    status = 'FAILED',
    error = 'Dead worker'
where status = 'RUNNING'
and dead_at < now()
returning id