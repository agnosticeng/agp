select 
    *
from agp_execution
where query_id = @query_id
and status = any(@statuses)
order by {{.sort_by}} desc
limit {{.limit}}
