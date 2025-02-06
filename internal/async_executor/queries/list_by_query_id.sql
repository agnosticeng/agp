select 
    *
from agp_execution
where query_id = @query_id
and status = any(@statuses)
{{if ne .query_hash "" }}
and query_hash = @query_hash
{{end}}
order by {{.sort_by}} desc
limit {{.limit}}
