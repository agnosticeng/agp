delete from agp_execution
where id = any(@ids)
returning id