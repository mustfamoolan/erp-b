-- Migration 000019: Granular permission catalogue
-- Replaces coarse "*.manage" permissions with view/create/update/... permissions,
-- gives every permission a proper Arabic display name, and re-maps existing roles
-- so no role loses an ability it had before.
-- Deletion permissions are intentionally absent (deactivate instead — AGENTS.md §1.13, §2.5.2).

-- ─── 1. Upsert full catalogue ───────────────────────────────────────────────
INSERT INTO permissions (code, display_name, description, grp) VALUES
    -- Users
    ('users.view',            'عرض المستخدمين',                 'عرض قائمة المستخدمين وتفاصيلهم',                     'users'),
    ('users.create',          'إضافة مستخدم',                   'إنشاء حساب مستخدم جديد',                             'users'),
    ('users.update',          'تعديل مستخدم',                   'تعديل بيانات مستخدم أو إيقافه',                      'users'),
    ('users.assign_roles',    'تعيين أدوار المستخدم',            'منح أو سحب الأدوار من المستخدم',                     'users'),
    ('users.assign_scopes',   'منح الوصول للمعامل',              'منح أو سحب صلاحية وصول المستخدم إلى معمل/جهة',        'users'),
    -- Roles
    ('roles.view',            'عرض الأدوار والصلاحيات',          'عرض الأدوار وقائمة الصلاحيات',                       'roles'),
    ('roles.create',          'إضافة دور',                      'إنشاء دور جديد وتحديد صلاحياته',                     'roles'),
    ('roles.update',          'تعديل صلاحيات دور',               'تعديل الصلاحيات الممنوحة لدور',                      'roles'),
    -- Factories & areas
    ('factories.view',        'عرض المعامل والجهات',             'عرض قائمة المعامل والجهات التنظيمية',                 'factories'),
    ('factories.create',      'إضافة معمل',                     'إنشاء معمل/جهة جديدة',                               'factories'),
    ('factories.update',      'تعديل معمل',                     'تعديل بيانات معمل أو إيقافه',                        'factories'),
    ('areas.view',            'عرض المناطق',                    'عرض قائمة المناطق',                                  'areas'),
    ('areas.create',          'إضافة منطقة',                    'إنشاء منطقة جديدة',                                  'areas'),
    ('areas.update',          'تعديل منطقة',                    'تعديل بيانات منطقة',                                 'areas'),
    -- Employees
    ('employees.view',        'عرض الموظفين',                   'عرض قائمة الموظفين وتفاصيلهم',                       'employees'),
    ('employees.create',      'إضافة موظف',                     'إنشاء موظف جديد',                                    'employees'),
    ('employees.update',      'تعديل موظف',                     'تعديل بيانات موظف أو إيقافه',                        'employees'),
    -- Master data
    ('master_data.view',      'عرض البيانات الأساسية',           'عرض أنواع الطلبات وتصنيفات الصرف',                   'master_data'),
    ('master_data.create',    'إضافة بيانات أساسية',             'إضافة نوع طلب أو تصنيف صرف',                         'master_data'),
    ('master_data.update',    'تعديل بيانات أساسية',             'تعديل نوع طلب أو تصنيف صرف',                         'master_data'),
    ('master_data.toggle',    'تفعيل/إيقاف بيانات أساسية',       'تفعيل أو إيقاف نوع طلب أو تصنيف صرف',                'master_data'),
    -- Exchange rates
    ('exchange_rate.view',    'عرض أسعار الصرف',                'عرض أسعار الصرف الحالية والسابقة',                   'exchange_rate'),
    ('exchange_rate.update',  'تحديث سعر الصرف',                'إدخال سعر صرف جديد',                                 'exchange_rate'),
    -- Accounting
    ('accounting.view',           'عرض الحسابات والقيود',        'عرض شجرة الحسابات والأرصدة والفترات',                'accounting'),
    ('accounting.journal_create', 'إنشاء قيد يومية',             'إنشاء قيد يومية يدوي (مسودة)',                       'accounting'),
    ('accounting.post',           'ترحيل القيود',                'ترحيل قيود اليومية',                                 'accounting'),
    ('accounting.reverse',        'عكس القيود',                  'إنشاء قيد عكسي لقيد مرحّل',                          'accounting'),
    ('accounting.close_period',   'إقفال الفترات المحاسبية',      'إقفال فترة محاسبية',                                 'accounting'),
    -- Cashboxes
    ('cashbox.view',            'عرض الصناديق',                 'عرض الصناديق وأرصدتها وحركاتها',                     'cashbox'),
    ('cashbox.create',          'إنشاء صندوق',                  'إنشاء صندوق جديد مع حسابه المحاسبي',                 'cashbox'),
    ('cashbox.opening_balance', 'إثبات رصيد افتتاحي',           'إثبات الرصيد الافتتاحي للصندوق (مرة واحدة)',         'cashbox'),
    ('cashbox.transfer',        'تحويل بين الصناديق',            'تحويل مبالغ بين الصناديق',                           'cashbox'),
    -- Expenses
    ('expenses.create',       'تسجيل مصروف',                    'تسجيل مصروف نقدي من صندوق المعمل',                   'expenses'),
    -- Inventory
    ('warehouse.view',        'عرض المخازن والأصناف',            'عرض المخازن والأصناف والوحدات',                      'warehouse'),
    ('warehouse.create',      'إنشاء مخزن',                     'إنشاء مخزن جديد',                                    'warehouse'),
    ('items.create',          'إضافة أصناف',                    'إضافة صنف أو تصنيف صنف أو متغير',                    'warehouse'),
    ('warehouse.receive',     'استلام مواد للمخزن',              'تسجيل حركة استلام/إدخال للمخزن',                     'warehouse'),
    ('warehouse.issue',       'صرف مواد من المخزن',              'تسجيل حركة صرف/استهلاك من المخزن',                   'warehouse'),
    ('warehouse.transfer',    'تحويل بين المخازن',               'نقل مواد بين المخازن',                               'warehouse'),
    ('warehouse.adjust',      'تسوية المخزون',                  'تسجيل تسوية جردية أو رصيد افتتاحي للمخزون',          'warehouse'),
    -- Requests
    ('request.view',          'عرض طلبات المعمل',                'عرض طلبات المعامل المسموح بها',                      'request'),
    ('request.view_all',      'عرض جميع الطلبات',               'عرض كل الطلبات الواردة للإدارة',                     'request'),
    ('request.create',        'إنشاء طلب مالي',                 'إنشاء طلب مالي جديد (مسودة)',                         'request'),
    ('request.submit',        'تقديم الطلب',                    'تقديم الطلب إلى التدقيق',                            'request'),
    ('request.cancel',        'إلغاء الطلب',                    'إلغاء طلب قبل اعتماده',                              'request'),
    ('request.attach',        'إرفاق مستندات للطلب',             'رفع وصولات ومرفقات (صور/PDF) للطلب',                 'request'),
    ('request.review',        'بدء تدقيق الطلب',                 'استلام الطلب للتدقيق',                               'request'),
    ('request.audit_approve', 'موافقة التدقيق',                  'الموافقة على الطلب بعد التدقيق',                     'request'),
    ('request.approve',       'الاعتماد المالي للطلب',            'الاعتماد المالي من محاسب الإدارة',                   'request'),
    ('request.reject',        'رفض الطلب',                      'رفض الطلب مع ذكر السبب',                             'request'),
    ('request.pay',           'صرف مبلغ الطلب',                  'تنفيذ دفع الطلب من صندوق الإدارة',                   'request'),
    ('request.receive',       'تأكيد استلام المبلغ',             'تأكيد استلام المعمل للمبلغ وإكمال الطلب',            'request'),
    -- Purchases
    ('purchases.view',        'عرض فواتير الشراء',               'عرض فواتير الشراء',                                  'purchases'),
    ('purchases.create',      'إنشاء فاتورة شراء',               'إنشاء وتعديل فاتورة شراء (مسودة)',                   'purchases'),
    ('purchases.approve',     'اعتماد فاتورة شراء',              'اعتماد فاتورة شراء',                                 'purchases'),
    ('purchases.pay',         'دفع فاتورة شراء',                 'تسجيل دفع فاتورة شراء من الصندوق',                   'purchases'),
    -- Audit & reports
    ('audit.view',            'عرض سجل التدقيق',                'عرض سجل العمليات (من، متى، ماذا)',                   'audit'),
    ('reports.view',          'عرض التقارير',                   'عرض التقارير المحاسبية والمخزنية',                   'reports'),
    -- Scope
    ('scope.all',             'الوصول لكل المعامل',              'رؤية بيانات جميع المعامل والجهات',                   'admin')
ON CONFLICT (code) DO UPDATE
    SET display_name = EXCLUDED.display_name,
        description  = EXCLUDED.description,
        grp          = EXCLUDED.grp;

-- ─── 2. Re-map roles: old permission → new granular permissions ─────────────
CREATE TEMP TABLE perm_map (old_code VARCHAR(100), new_code VARCHAR(100));
INSERT INTO perm_map (old_code, new_code) VALUES
    ('users.manage',     'users.view'),
    ('users.manage',     'users.create'),
    ('users.manage',     'users.update'),
    ('users.manage',     'users.assign_roles'),
    ('users.manage',     'users.assign_scopes'),
    ('users.manage',     'exchange_rate.update'),   -- exchange rate POST was guarded by users.manage
    ('roles.manage',     'roles.view'),
    ('roles.manage',     'roles.create'),
    ('roles.manage',     'roles.update'),
    ('factories.manage', 'factories.create'),
    ('factories.manage', 'factories.update'),
    ('factories.manage', 'areas.create'),
    ('factories.manage', 'areas.update'),
    ('employees.manage', 'employees.view'),
    ('employees.manage', 'employees.create'),
    ('employees.manage', 'employees.update'),
    ('request.create',   'request.submit'),
    ('request.create',   'request.cancel'),
    ('request.create',   'request.attach'),
    ('request.review',   'request.view_all'),
    ('request.review',   'request.audit_approve'),
    ('accounting.post',  'accounting.journal_create'),
    ('warehouse.adjust', 'warehouse.create'),
    ('warehouse.adjust', 'items.create');

INSERT INTO role_permissions (role_id, permission_id)
SELECT DISTINCT rp.role_id, pn.id
FROM role_permissions rp
JOIN permissions po ON po.id = rp.permission_id
JOIN perm_map m     ON m.old_code = po.code
JOIN permissions pn ON pn.code = m.new_code
ON CONFLICT (role_id, permission_id) DO NOTHING;

DROP TABLE perm_map;

-- ─── 3. Read access that used to be open to any authenticated user ─────────
-- (factories/areas/master data/exchange rates lists are needed by every screen header & form)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('SUPER_ADMIN','ADMINISTRATOR','AUDITOR','ACCOUNTANT','FACTORY_ACCOUNTANT','FACTORY_EMPLOYEE')
  AND p.code IN ('factories.view','areas.view','master_data.view','exchange_rate.view')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ─── 4. Role defaults for permissions that had no previous equivalent ──────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON
     (r.name = 'ACCOUNTANT'         AND p.code IN ('purchases.view','purchases.create','purchases.approve','purchases.pay','expenses.create'))
  OR (r.name = 'FACTORY_ACCOUNTANT' AND p.code IN ('purchases.view','purchases.create','purchases.pay','expenses.create'))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- SUPER_ADMIN and ADMINISTRATOR hold the full catalogue.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name IN ('SUPER_ADMIN','ADMINISTRATOR')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ─── 5. Retire the coarse permissions (role_permissions cascade) ────────────
DELETE FROM permissions WHERE code IN ('users.manage','roles.manage','factories.manage','employees.manage');
