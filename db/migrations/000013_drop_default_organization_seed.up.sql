-- 弱化历史 default organization 种子。
-- 仅在没有任何成员、项目、邀请引用时才删除，避免误伤已有数据。
DELETE FROM organizations
WHERE id = '00000000-0000-0000-0000-000000000001'
  AND NOT EXISTS (
      SELECT 1 FROM organization_members WHERE organization_id = '00000000-0000-0000-0000-000000000001'
  )
  AND NOT EXISTS (
      SELECT 1 FROM projects WHERE organization_id = '00000000-0000-0000-0000-000000000001'
  )
  AND NOT EXISTS (
      SELECT 1 FROM organization_invitations WHERE organization_id = '00000000-0000-0000-0000-000000000001'
  );
