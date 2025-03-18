-- migrations/003_create_trace_table.sql
CREATE TABLE webapp.traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES webapp.users(id),
    instructor_id UUID REFERENCES webapp.instructors(id),
    course_id UUID REFERENCES webapp.courses(id),
    status VARCHAR(10) NOT NULL CHECK (status IN ('failed', 'processed', 'uploaded')),
    vector_id VARCHAR(100),
    file_name VARCHAR(255) NOT NULL,
    bucket_url TEXT NOT NULL,
    date_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);