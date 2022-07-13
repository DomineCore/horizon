alter table tb_template add column repository varchar(256) not null default '' after `description`;
alter table tb_template add column token varchar(256) not null default '' after `repository`;
alter table tb_template add column group_id bigint(20) unsigned not null default 0 after `token`;
alter table tb_template add column chart_name varchar(256) default '' after `name`;
alter table tb_template_release add column template bigint(20) unsigned not null default 0 after `template_name`;
alter table tb_template_release add column chart_name varchar(256) not null default '' after `name`;
update tb_template set chart_name = `name`;
update tb_template set group_id = 0 where 1;
update tb_template_release a, tb_template b set a.template = b.id where a.template_name = b.name;
update tb_template_release a, tb_template b set a.chart_name = b.chart_name where a.template = b.id;
alter table tb_template add column only_admin bool not null default false after group_id;
alter table tb_template_release add column only_admin bool not null default false after recommended;
