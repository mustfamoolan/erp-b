DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('cashbox.create', 'cashbox.opening_balance', 'scope.all')
);

DELETE FROM permissions WHERE code IN ('cashbox.create', 'cashbox.opening_balance', 'scope.all');
