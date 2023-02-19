CREATE TABLE workout (
    id            integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    started_utc_s integer NOT NULL
);

CREATE TABLE exercise (
    id   integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    name text    NOT NULL
);

CREATE TABLE exercise_set (
    id            integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    exercise_id   integer NOT NULL,
    workout_id    integer NOT NULL,
    created_utc_s integer NOT NULL,
    repetitions   integer NOT NULL,
    weight        integer NOT NULL,
    note          text,

    FOREIGN KEY (exercise_id) REFERENCES exercise (id),
    FOREIGN KEY (workout_id) REFERENCES workout (id) ON DELETE CASCADE
);

INSERT INTO exercise (name)
VALUES ('Dehnen'),
       ('Handstand'),
       ('Squats'),
       ('Deadlifts'),
       ('Schulterdrücken Langhantel'),
       ('Schulterdrücken Maschine'),
       ('Seitheben Maschine'),
       ('Bankdrücken'),
       ('Bizeps Maschine'),
       ('Butterfly Maschine'),
       ('Reverse Butterfly Maschine'),
       ('Muscle Up'),
       ('Front Lever'),
       ('Back Lever'),
       ('Human Flag'),
       ('Pull Up'),
       ('Lat Pull-Down (Turm)'),
       ('Rudern (Turm)'),
       ('Dips'),
       ('Beinstrecken'),
       ('Beinpresse'),
       ('Wadenheben'),
       ('Adduktoren Maschine (Muskeln Innenseite)'),
       ('Abduktoren Maschine (Muskeln Außenseite)');