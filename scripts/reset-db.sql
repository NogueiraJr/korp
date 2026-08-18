-- ============================================================================
-- reset-db.sql
-- Apaga todas as linhas de todas as tabelas do schema 'public' em cascata.
-- Atenção: reinicia os SERIALs (IDs reiniciam em 1).
-- ============================================================================
--
-- Este script deve ser executado contra cada banco que se deseja limpar:
--   psql -U korp -d korp_estoque -f scripts/reset-db.sql
--   psql -U korp -d korp_faturamento -f scripts/reset-db.sql
--
-- O uso de CASCADE garante que asForeign Keys são respeitadas (apaga
-- tabelas filhas antes dos pais), mesmo que a ordem seja desconhecida.

DO $$
DECLARE
    t record;
BEGIN
    -- Itera sobre todas as tabelas do schema public
    FOR t IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename <> 'alembic_meta'  -- caso o projeto use Alembic
          AND tablename NOT LIKE 'gis_%'   -- excluir extensões/geografia se houver
    LOOP
        -- TRUNCATE ... CASCADE apaga a tabela e todas as que a referenciam via FK
        EXECUTE 'TRUNCATE ' || quote_ident(t.tablename) || ' CASCADE';
    END LOOP;
END$$;