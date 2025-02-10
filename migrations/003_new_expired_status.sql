-- enum-like contraint
alter table agp_execution drop constraint const_agp_execution_status;

alter table agp_execution add constraint const_agp_execution_status 
check (
  status in (
    'PENDING',
    'RUNNING',
    'CANCELED',
    'FAILED',
    'SUCCEEDED',
    'EXPIRED'
  )
) not valid;

-- used to implement GC
create index idx_agp_execution_query_status_expired
on agp_execution (status)
where status = 'EXPIRED';

-- allowed status transitions

-- PENDING    -> (PENDING, RUNNING, CANCELED)
-- RUNNING    -> (PENDING, RUNNING, SUCCEEDED, FAILED, CANCELED)
-- CANCELED   -> (CANCELED, EXPIRED)
-- SUCCEEDED  -> (SUCCEEDED, EXPIRED)
-- FAILED     -> (FAILED, EXPIRED)
-- EXPIRED    -> (EXPIRED)

create or replace function fn_agp_execution_validate_status_update()
returns trigger
as $$
  begin
    case old.status
      when 'PENDING' then 
        case new.status
          when 'PENDING' then null;
          when 'RUNNING' then null;
          when 'CANCELED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'RUNNING' then
        case new.status
          when 'PENDING' then null;
          when 'RUNNING' then null;
          when 'SUCCEEDED' then null;
          when 'FAILED' then null;
          when 'CANCELED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'CANCELED' then
        case new.status
          when 'CANCELED' then null;
          when 'EXPIRED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'SUCCEEDED' then
        case new.status
          when 'SUCCEEDED' then null;
          when 'EXPIRED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'FAILED' then
        case new.status
          when 'FAILED' then null;
          when 'EXPIRED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'EXPIRED' then
        case new.status
          when 'EXPIRED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
    end case;
    return new;
  end;
$$ language plpgsql;