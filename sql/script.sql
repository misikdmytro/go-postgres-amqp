CREATE EXTENSION IF NOT EXISTS amqp;
INSERT INTO amqp.broker (host, port, vhost, username, password)
VALUES ('rabbitmq3', 5672, '/', 'guest', 'guest')
RETURNING broker_id;
CREATE TABLE IF NOT EXISTS users (id serial PRIMARY KEY, name text);
CREATE OR REPLACE FUNCTION notify_insert() RETURNS trigger AS $notify_insert_trigger$
DECLARE payload text;
BEGIN payload = json_build_object('new', row_to_json(NEW))::text;
PERFORM amqp.publish(1, 'pg', 'insert', payload);
RETURN NEW;
END;
$notify_insert_trigger$ LANGUAGE plpgsql;
CREATE OR REPLACE TRIGGER notify_insert_trigger
AFTER
INSERT ON users FOR EACH ROW EXECUTE PROCEDURE notify_insert();
CREATE OR REPLACE FUNCTION notify_update() RETURNS trigger AS $notify_update_trigger$
DECLARE payload text;
BEGIN payload = json_build_object('old', row_to_json(OLD), 'new', row_to_json(NEW))::text;
PERFORM amqp.publish(1, 'pg', 'update', payload);
RETURN NEW;
END;
$notify_update_trigger$ LANGUAGE plpgsql;
CREATE OR REPLACE TRIGGER notify_update_trigger
AFTER
UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE notify_update();
CREATE OR REPLACE FUNCTION notify_delete() RETURNS trigger AS $notify_delete_trigger$
DECLARE payload text;
BEGIN payload = json_build_object('old', row_to_json(OLD))::text;
PERFORM amqp.publish(1, 'pg', 'delete', payload);
RETURN OLD;
END;
$notify_delete_trigger$ LANGUAGE plpgsql;
CREATE OR REPLACE TRIGGER notify_delete_trigger
AFTER DELETE ON users FOR EACH ROW EXECUTE PROCEDURE notify_delete();
INSERT INTO users (name)
VALUES ('John Doe');
UPDATE users
SET name = 'Jane Doe'
WHERE name = 'John Doe';
DELETE FROM users
WHERE name = 'Jane Doe';