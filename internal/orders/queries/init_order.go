package queries

var InsertOrderQuery = `INSERT INTO orders (tid, id, reference, currency, total, updated_at) 
		      VALUES (@tid, @id, @reference, @currency, @total, NOW())`

var InsertCustomerQuery = `INSERT INTO customers (tid, id, email, updated_at) 
		      VALUES (@tid, @id, @email, NOW())`
