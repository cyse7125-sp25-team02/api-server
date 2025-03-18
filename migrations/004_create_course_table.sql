-- migrations/004_create_course_table.sql
CREATE TABLE webapp.courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    semester_term VARCHAR(10) NOT NULL CHECK (semester_term IN ('Fall', 'Spring', 'Summer')),
    credit_hours INTEGER NOT NULL CHECK (credit_hours > 0),
    subject_code VARCHAR(10) NOT NULL, -- e.g., INFO, DAMG, CSYE
    course_id INTEGER NOT NULL CHECK (course_id BETWEEN 1 AND 99999999), -- Up to 8 digits
    semester_year INTEGER NOT NULL CHECK (semester_year >= 2000), -- Reasonable year validation
    date_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_id UUID REFERENCES webapp.users(id),
    instructor_id UUID REFERENCES webapp.instructors(id)
);
