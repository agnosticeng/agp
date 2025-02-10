select 
    *
from agp_execution
where status = 'EXPIRED'
limit @limit
