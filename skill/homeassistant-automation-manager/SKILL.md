# SKILL.md
---
name: homeassistant-automation-manager
description: 管理Home Assistant自动化配置，支持查询现有自动化列表、新增实体状态触发的自动化、删除指定自动化。Use when用户提及"Home Assistant自动化"、"查询HA自动化列表"、"新增Home Assistant自动化"、"删除HA自动化"、"创建HA实体触发自动化"。
metadata:
  version: 1.0.0
  category: homeassistant
  emoji:"🏠"
  requires: ["jq","curl"]
  tags: [智能家居, 自动化管理, homeassistant]
---

# Home Assistant Automation Manager
## 核心指令
本技能用于管理Home Assistant的自动化配置，核心支持**查询自动化列表**、**新增实体状态触发自动化**、**删除指定自动化**三大功能。
## Setup

### Option 1: Config File (Recommended)

Create `~/.config/home-assistant/config.json`:
```json
{
  "url": "https://your-ha-instance.duckdns.org",
  "token": "your-long-lived-access-token"
}
```

### Option 2: Environment Variables

```bash
export HA_URL="http://homeassistant.local:8123"
export HA_TOKEN="your-long-lived-access-token"
```

### Getting a Long-Lived Access Token

1. Open Home Assistant → Profile (bottom left)
2. Scroll to "Long-Lived Access Tokens"
3. Click "Create Token", name it (e.g., "Clawdbot")
4. Copy the token immediately (shown only once)


### 功能1: 查询现有自动化列表
#### 执行步骤
1. 调用技能脚本的`ha.sh get-automation-list`方法，获取所有自动化的完整信息（名称、ID、实体ID、状态、上次触发时间）。
2. 对返回结果进行格式化整理，提取核心字段（名称、ID、实体ID、状态、上次触发时间），剔除冗余配置。
3. 以清晰的列表形式向用户展示，标注禁用的自动化并说明。

#### 预期输出
按序号排列的自动化列表，每条包含**自动化名称**、**唯一ID**、**触发条件**、**执行动作**、**启用状态**，示例：
```
1. 名称：厨房人体感应开灯 | ID：1774070795 | 实体：automation.chufang_renti_ganying_kaideng | 上一次触发时间：2026-03-24 08:03:58 | 状态：启用
2. 名称：离家关闭所有灯   |ID：1774070796 | 实体：automation.away_turn_off_lights | 上一次触发时间：2026-03-24 08:03:58|  状态：禁用
```

### 功能2: 新增实体状态触发的自动化
#### 核心要求
必须向用户获取**3个关键信息**，缺失时主动询问，无默认值：
1. 自动化名称（唯一，不可与现有重复）；
//这里需要修改，为设备名称变为某个状态的时候触发自动化
2. 触发条件：严格遵循`[实体ID]变为[目标状态]`格式（如`sensor.motion_living变为on`、`binary_sensor.door_main变为off`）；
3. 执行动作：自然语言描述的智能家居操作（如"打开客厅的灯"、"关闭卧室空调"、"启动扫地机器人"）。

#### 执行步骤
1. 校验用户提供的触发条件格式是否合规，若不合规提示用户按`[实体ID]变为[目标状态]`重新输入；
2. 调用Home Assistant MCP工具的`check_entity_exists`接口，验证触发条件中的实体ID是否存在，不存在则提示用户核对实体ID；
3. 校验自动化名称是否与现有重复，重复则提示用户修改名称；
4. 将自然语言的执行动作转换为Home Assistant可执行的服务调用指令（基于MCP工具的`convert_nl_to_service`接口）；
5. 调用Home Assistant MCP工具的`create_automation`接口，传入**名称、触发条件（实体状态变化）、执行动作（服务指令）**，创建自动化并启用；
6. 向用户返回创建结果，包含自动化名称、唯一ID及完整配置，提示可立即测试。

#### 格式校验规则
- 触发条件必须包含**实体ID**和**变为**关键词，目标状态为Home Assistant标准状态值（on/off、home/not_home、open/closed等）；
- 自动化名称不可包含特殊字符（如/、\、:、*），长度控制在5-30个字符。

### 功能5: 删除指定自动化
#### 执行步骤
1. 若用户仅说明"删除自动化"，先调用`get_automations`接口获取列表，让用户选择需删除的**自动化名称或唯一ID**；
2. 若用户已提供名称/ID，调用get_automations接口验证id是否存在；
3. 向用户二次确认是否删除（防止误操作）；
4. 确认后调用`delete_automation`接口，根据唯一ID执行删除操作；
5. 向用户返回删除成功提示，若删除失败说明原因。

## 操作示例
### 示例1: 查询自动化列表
用户输入："帮我查询Home Assistant的自动化列表"
执行动作：调用`get_automations`接口，格式化返回所有自动化信息
输出结果：按核心指令的预期输出格式展示列表

### 示例2: 新增自动化
用户输入："创建一个HA自动化，名称是阳台门打开开窗帘，触发条件是binary_sensor.door_balcony变为on，动作是打开阳台窗帘"
执行动作：
1. 校验格式：触发条件合规，名称无特殊字符；
2. 验证实体`binary_sensor.door_balcony`存在；
3. 转换动作"打开阳台窗帘"为HA服务指令；
4. 调用`create_automation`创建并启用；
输出结果："自动化「阳台门打开开窗帘」创建成功，唯一ID：automation.balcony_door_curtain，触发条件：binary_sensor.door_balcony变为on，执行动作：打开阳台窗帘，当前状态：启用。"

### 示例3: 删除自动化
用户输入："删除HA里的厨房人体感应开灯自动化"
执行动作：
1. 验证自动化存在；
2. 二次确认："是否确认删除自动化「厨房人体感应开灯」（ID：automation.kitchen_motion_light）？"；
3. 用户确认后执行删除；
输出结果："自动化「厨房人体感应开灯」（ID：automation.kitchen_motion_light）已成功删除。"

## 故障排除
### 错误1: 触发条件格式错误
**错误提示**：触发条件格式不符合要求
**解决方案**：提示用户按`[实体ID]变为[目标状态]`格式输入，示例：`sensor.motion_bath变为on`。

### 错误2: 实体ID不存在
**错误提示**：触发条件中的实体[XXX]不存在于Home Assistant
**解决方案**：让用户在Home Assistant的「开发者工具-状态」中核对正确的实体ID，重新输入。

### 错误3: 自动化名称重复
**错误提示**：自动化名称[XXX]已存在，不可重复创建
**解决方案**：提示用户修改自动化名称，建议增加个性化标识（如区域、功能）。

### 错误4: 执行动作转换失败
**错误提示**：无法将自然语言动作转换为Home Assistant服务指令
**解决方案**：让用户更清晰地描述动作，明确操作的设备名称/实体，示例：将"开空调"改为"打开主卧的空调"。

### 错误5: 自动化不存在（删除时）
**错误提示**：未找到指定名称/ID的自动化
**解决方案**：先为用户查询现有自动化列表，让用户选择正确的名称/ID后重新执行删除操作。