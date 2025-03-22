// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import PropTypes from 'prop-types';
import React from 'react';

export default class IncidentPostTypeComponent extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
    };

    constructor(props) {
        super(props);
        this.state = {};
    }

    getStyle = () => {
        const {theme} = this.props;

        return {
            container: {
                display: 'flex',
                flexDirection: 'column',
                padding: '12px',
                border: '1px solid #ddd',
                margin: '5px 0',
                borderRadius: '4px',
                backgroundColor: theme.centerChannelBg,
            },
            header: {
                display: 'flex',
                flexDirection: 'row',
                alignItems: 'center',
                marginBottom: '10px',
            },
            title: {
                fontWeight: 'bold',
                fontSize: '16px',
                color: theme.centerChannelColor,
                marginLeft: '10px',
            },
            icon: {
                height: '20px',
                width: '20px',
            },
            content: {
                color: theme.centerChannelColor,
            },
            footer: {
                marginTop: '10px',
                borderTop: '1px solid #ddd',
                paddingTop: '10px',
                fontSize: '12px',
                color: theme.centerChannelColor,
            },
            button: {
                border: '1px solid',
                borderRadius: '4px',
                fontWeight: 'bold',
                padding: '6px 12px',
                margin: '0 5px',
                cursor: 'pointer',
                fontSize: '12px',
            },
            acknowledgeButton: {
                borderColor: theme.buttonBg,
                backgroundColor: theme.buttonBg,
                color: theme.buttonColor,
            },
            resolveButton: {
                borderColor: '#4caf50',
                backgroundColor: '#4caf50',
                color: 'white',
            },
            incidentDetails: {
                marginBottom: '10px',
                display: 'flex',
                flexDirection: 'column',
            },
            detailRow: {
                marginBottom: '5px',
            },
            detailLabel: {
                fontWeight: 'bold',
                marginRight: '5px',
            },
            buttonContainer: {
                display: 'flex',
                flexDirection: 'row',
                marginTop: '10px',
            },
        };
    };

    render() {
        const {post} = this.props;
        const style = this.getStyle();

        if (!post.props || !post.props.attachments || !post.props.attachments.length) {
            return null;
        }

        const attachment = post.props.attachments[0];

        if (!attachment.callback_id) {
            return null;
        }

        const fields = attachment.fields || [];

        const fieldMap = fields.reduce((map, field) => {
            map[field.title] = field.value;
            return map;
        }, {});

        return (
            <div style={style.container}>
                <div style={style.header}>
                    <img
                        style={style.icon}
                        src='/plugins/com.github.mnzsyu.mattermost-pagerduty-plugin/public/pagerduty-icon.svg'
                        alt='PagerDuty Logo'
                    />
                    <div style={style.title}>{attachment.title || 'PagerDuty Incident'}</div>
                </div>
                <div style={style.content}>
                    <div style={style.incidentDetails}>
                        {Object.entries(fieldMap).map(([title, value], index) => (
                            <div
                                key={index}
                                style={style.detailRow}
                            >
                                <span style={style.detailLabel}>{title}{':'}</span>
                                <span>{value}</span>
                            </div>
                        ))}
                    </div>
                    <div>{attachment.text}</div>
                </div>
                {attachment.actions && attachment.actions.length > 0 && (
                    <div style={style.buttonContainer}>
                        {attachment.actions.map((action, index) => {
                            let buttonStyle = style.button;
                            if (action.id === 'acknowledge') {
                                buttonStyle = {...style.button, ...style.acknowledgeButton};
                            } else if (action.id === 'resolve') {
                                buttonStyle = {...style.button, ...style.resolveButton};
                            }

                            if (action.type === 'select') {
                                return null;
                            }

                            return (
                                <button
                                    key={index}
                                    style={buttonStyle}
                                    onClick={() => {
                                    }}
                                >
                                    {action.name}
                                </button>
                            );
                        })}
                    </div>
                )}
                <div style={style.footer}>
                    {'mnzsyu/mattermost-pagerduty-plugin'}
                </div>
            </div>
        );
    }
}
