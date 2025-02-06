-- Create controller leader election table

create table agp_lease (
    id bigserial primary key,
    key text not null,
    owner text not null,
    end_of_term timestamp with time zone not null 
);

alter table agp_lease
add constraint agp_lease_key_key
unique (key);

---- create above / drop below ----

drop table agp_lease;
