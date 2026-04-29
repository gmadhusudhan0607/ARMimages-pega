-- CREATE EXTENSION IF NOT EXISTS vector;
DO
$$
    BEGIN
        CREATE EXTENSION IF NOT EXISTS vector;
    EXCEPTION
        WHEN unique_violation THEN NULL;
    END;
$$ LANGUAGE plpgsql;

DO
$$
    BEGIN
        ALTER EXTENSION vector UPDATE;
    EXCEPTION
        WHEN unique_violation THEN NULL;
    END;
$$ LANGUAGE plpgsql;