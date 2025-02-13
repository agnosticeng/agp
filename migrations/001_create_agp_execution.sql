-- Create base agp execution tables

create table agp_execution (
  id bigserial primary key,
  created_at timestamp with time zone not null default now(),
  created_by text not null,
  query_id text not null,
  query_hash text not null,
  query text not null,
  tier text not null,
  status text not null,
  collapsed_counter integer not null default 0,
  picked_at timestamp with time zone,
  picked_by text,
  progress jsonb,
  dead_at timestamp with time zone,
  completed_at timestamp with time zone,
  result jsonb,
  error text
);

-- enum-like constraint
alter table agp_execution add constraint const_agp_execution_status 
check (
  status in (
    'PENDING',
    'RUNNING',
    'CANCELED',
    'FAILED',
    'SUCCEEDED'
  )
);

-- ensures we can find execution for a given query_id fast
create index idx_agp_execution_query_id
on agp_execution (query_id);

-- used to implement query collapsing as an upsert
create unique index idx_agp_execution_query_id_status
on agp_execution (query_id)
where status in ('PENDING', 'RUNNING');

-- ensures we can find dead executions fast
create index idx_agp_execution_dead_at
on agp_execution (dead_at)
where status = 'RUNNING';

-- ensures picking a PENDING job is fast
create index idx_agp_execution_tier
on agp_execution (tier)
where status = 'PENDING';

-- allowed status transitions

-- PENDING    -> (PENDING, RUNNING, CANCELED)
-- RUNNING    -> (PENDING, RUNNING, SUCCEEDED, FAILED, CANCELED)
-- CANCELED   -> (CANCELED)
-- SUCCEEDED  -> (SUCCEEDED)
-- FAILED     -> (FAILED)

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
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'SUCCEEDED' then
        case new.status
          when 'SUCCEEDED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
      when 'FAILED' then
        case new.status
          when 'FAILED' then null;
          else raise exception 'Bad status transition: % --> %', old.status, new.status;
        end case;
    end case;
    return new;
  end;
$$ language plpgsql;

create trigger trgr_before_update_agp_execution_status 
before update of status on agp_execution
for each row
execute function fn_agp_execution_validate_status_update();

---- create above / drop below ----

drop table agp_execution;
drop function agp_validate_status_transition;
