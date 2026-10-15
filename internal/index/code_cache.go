package index

import "database/sql"

func (d *DB) GetCodeCache(path, hash string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var data []byte
	err := d.sqlDB.QueryRow("SELECT data FROM code_cache WHERE path=? AND hash=?", path, hash).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return data, err
}
func (d *DB) PutCodeCache(path, hash string, data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.sqlDB.Exec("INSERT INTO code_cache(path,hash,data) VALUES(?,?,?) ON CONFLICT(path) DO UPDATE SET hash=excluded.hash,data=excluded.data", path, hash, data)
	return err
}
