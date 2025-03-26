CREATE TABLE places (
    id SERIAL PRIMARY KEY,         
    name VARCHAR(255) NOT NULL,      
    description TEXT,               
    latitude DOUBLE PRECISION NOT NULL,  
    longitude DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

